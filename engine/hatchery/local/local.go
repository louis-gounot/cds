package local

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

// New instanciates a new hatchery local
func New() *HatcheryLocal {
	s := new(HatcheryLocal)
	s.Router = &api.Router{
		Mux: mux.NewRouter(),
	}
	s.LocalWorkerRunner = new(localWorkerRunner)
	return s
}

func (h *HatcheryLocal) Init(config interface{}) (cdsclient.ServiceConfig, error) {
	var cfg cdsclient.ServiceConfig
	sConfig, ok := config.(HatcheryConfiguration)
	if !ok {
		return cfg, sdk.WithStack(fmt.Errorf("invalid local hatchery configuration"))
	}

	cfg.Host = sConfig.API.HTTP.URL
	cfg.Token = sConfig.API.Token
	cfg.InsecureSkipVerifyTLS = sConfig.API.HTTP.Insecure
	cfg.RequestSecondsTimeout = sConfig.API.RequestTimeout
	return cfg, nil
}

// ApplyConfiguration apply an object of type HatcheryConfiguration after checking it
func (h *HatcheryLocal) ApplyConfiguration(cfg interface{}) error {
	if err := h.CheckConfiguration(cfg); err != nil {
		return err
	}

	var ok bool
	h.Config, ok = cfg.(HatcheryConfiguration)
	if !ok {
		return fmt.Errorf("Invalid configuration")
	}

	genname := h.Configuration().Name
	h.Common.Common.ServiceName = genname
	h.Common.Common.ServiceType = services.TypeHatchery
	h.HTTPURL = h.Config.URL
	h.MaxHeartbeatFailures = h.Config.API.MaxHeartbeatFailures
	var err error
	h.Common.Common.PrivateKey, err = jwt.ParseRSAPrivateKeyFromPEM([]byte(h.Config.RSAPrivateKey))
	if err != nil {
		return fmt.Errorf("unable to parse RSA private Key: %v", err)
	}

	return nil
}

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (h *HatcheryLocal) Status(ctx context.Context) sdk.MonitoringStatus {
	m := h.CommonMonitoring()
	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{
		Component: "Workers",
		Value:     fmt.Sprintf("%d/%d", len(h.WorkersStarted(ctx)), h.Config.Provision.MaxWorker),
		Status:    sdk.MonitoringStatusOK,
	})

	return m
}

// CheckConfiguration checks the validity of the configuration object
func (h *HatcheryLocal) CheckConfiguration(cfg interface{}) error {
	hconfig, ok := cfg.(HatcheryConfiguration)
	if !ok {
		return fmt.Errorf("Invalid configuration")
	}

	if hconfig.API.HTTP.URL == "" {
		return fmt.Errorf("API HTTP(s) URL is mandatory")
	}

	if hconfig.API.Token == "" {
		return fmt.Errorf("API Token URL is mandatory")
	}

	if hconfig.Basedir == "" {
		return fmt.Errorf("Invalid basedir directory")
	}

	if hconfig.Name == "" {
		return fmt.Errorf("please enter a name in your local hatchery configuration")
	}

	if ok, err := sdk.DirectoryExists(hconfig.Basedir); !ok {
		return fmt.Errorf("Basedir doesn't exist")
	} else if err != nil {
		return fmt.Errorf("Invalid basedir: %v", err)
	}
	return nil
}

// Serve start the hatchery server
func (h *HatcheryLocal) Serve(ctx context.Context) error {
	h.BasedirDedicated = filepath.Dir(filepath.Join(h.Config.Basedir, h.Configuration().Name))
	if ok, err := sdk.DirectoryExists(h.BasedirDedicated); !ok {
		log.Debug("creating directory %s", h.BasedirDedicated)
		if err := os.MkdirAll(h.BasedirDedicated, 0700); err != nil {
			return sdk.WrapError(err, "error while creating directory %s", h.BasedirDedicated)
		}
	} else if err != nil {
		return fmt.Errorf("Invalid basedir: %v", err)
	}

	if err := h.downloadWorker(); err != nil {
		return fmt.Errorf("Cannot download worker binary from api: %v", err)
	}

	return h.CommonServe(ctx, h)
}

func (h *HatcheryLocal) downloadWorker() error {
	urlBinary := h.Client.DownloadURLFromAPI("worker", sdk.GOOS, sdk.GOARCH, "")

	log.Debug("Downloading worker binary from %s", urlBinary)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	body, headers, _, err := h.Client.(cdsclient.Raw).Request(ctx, http.MethodGet, urlBinary, nil)
	if err != nil {
		return sdk.WrapError(err, "error while getting binary from CDS API")
	}

	if contentType := headers.Get("Content-Type"); contentType != "application/octet-stream" {
		return fmt.Errorf("invalid Binary (Content-Type: %s). Please try again or download it manually from %s", contentType, sdk.URLGithubReleases)
	}

	workerFullPath := path.Join(h.BasedirDedicated, h.getWorkerBinaryName())

	if _, err := os.Stat(workerFullPath); err == nil {
		log.Debug("removing existing worker binary from %s", workerFullPath)
		if err := os.Remove(workerFullPath); err != nil {
			return sdk.WrapError(err, "error while removing existing worker binary %s", workerFullPath)
		}
	}

	log.Debug("copy worker binary into %s", workerFullPath)
	fp, err := os.OpenFile(workerFullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0700)
	if err != nil {
		return sdk.WithStack(err)
	}

	if _, err := fp.Write(body); err != nil {
		return sdk.WithStack(err)
	}

	return sdk.WithStack(fp.Close())
}

func (h *HatcheryLocal) getWorkerBinaryName() string {
	workerName := "worker"

	if sdk.GOOS == "windows" {
		workerName += ".exe"
	}
	return workerName
}

// checkCapabilities checks all requirements, foreach type binary, check if binary is on current host
// returns an error "Exit status X" if current host misses one requirement
func (h *HatcheryLocal) checkCapabilities(req []sdk.Requirement) ([]sdk.Requirement, error) {
	var tmp map[string]sdk.Requirement

	tmp = make(map[string]sdk.Requirement)
	for _, r := range req {
		ok, err := h.checkRequirement(r)
		if err != nil {
			return nil, err
		}

		if ok {
			tmp[r.Name] = r
		}
	}

	capa := make([]sdk.Requirement, 0, len(tmp))
	for _, r := range tmp {
		capa = append(capa, r)
	}

	return capa, nil
}

//Configuration returns Hatchery CommonConfiguration
func (h *HatcheryLocal) Configuration() service.HatcheryCommonConfiguration {
	return h.Config.HatcheryCommonConfiguration
}

// CanSpawn return wether or not hatchery can spawn model.
// requirements are not supported
func (h *HatcheryLocal) CanSpawn(ctx context.Context, _ *sdk.Model, jobID int64, requirements []sdk.Requirement) bool {
	for _, r := range requirements {
		ok, err := h.checkRequirement(r)
		if err != nil || !ok {
			log.Debug("CanSpawn false hatchery.checkRequirement ok:%v err:%v r:%v", ok, err, r)
			return false
		}
	}

	for _, r := range requirements {
		if r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement {
			log.Debug("CanSpawn false service or memory")
			return false
		}

		if r.Type == sdk.OSArchRequirement && r.Value != (runtime.GOOS+"/"+runtime.GOARCH) {
			log.Debug("CanSpawn> job %d cannot spawn on this OSArch.", jobID)
			return false
		}
	}
	log.Debug("CanSpawn true for job %d", jobID)
	return true
}

// killWorker kill a local process
func (h *HatcheryLocal) killWorker(name string, workerCmd workerCmd) error {
	log.Info("KillLocalWorker> Killing %s", name)
	return workerCmd.cmd.Process.Kill()
}

// WorkersStarted returns the number of instances started but
// not necessarily register on CDS yet
func (h *HatcheryLocal) WorkersStarted(ctx context.Context) []string {
	h.Mutex.Lock()
	defer h.Mutex.Unlock()
	workers := make([]string, len(h.workers))
	var i int
	for n := range h.workers {
		workers[i] = n
		i++
	}
	return workers
}

// InitHatchery register local hatchery with its worker model
func (h *HatcheryLocal) InitHatchery(ctx context.Context) error {
	h.workers = make(map[string]workerCmd)
	sdk.GoRoutine(context.Background(), "startKillAwolWorkerRoutine", h.startKillAwolWorkerRoutine)
	return nil
}

func (h *HatcheryLocal) localWorkerIndexCleanup() {
	h.Lock()
	defer h.Unlock()

	needToDeleteWorkers := []string{}
	for name, workerCmd := range h.workers {
		// check if worker is still alive
		if workerCmd.cmd.ProcessState != nil && workerCmd.cmd.ProcessState.Exited() {
			log.Debug("process %s has been removed", name)
			needToDeleteWorkers = append(needToDeleteWorkers, name)
		}
	}

	for _, name := range needToDeleteWorkers {
		delete(h.workers, name)
	}
}

func (h *HatcheryLocal) startKillAwolWorkerRoutine(ctx context.Context) {
	t := time.NewTicker(5 * time.Second)
	for range t.C {
		if err := h.killAwolWorkers(); err != nil {
			log.Warning("Cannot kill awol workers: %s", err)
		}
	}
}

func (h *HatcheryLocal) killAwolWorkers() error {
	h.localWorkerIndexCleanup()

	h.Lock()
	defer h.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	apiWorkers, err := h.CDSClient().WorkerList(ctx)
	if err != nil {
		return err
	}

	mAPIWorkers := make(map[string]sdk.Worker, len(apiWorkers))
	for _, w := range apiWorkers {
		mAPIWorkers[w.Name] = w
	}

	killedWorkers := []string{}
	for name, workerCmd := range h.workers {
		var kill bool
		// if worker not found on api side or disabled, kill it
		if w, ok := mAPIWorkers[name]; !ok {
			// if no name on api, and worker create less than 10 seconds, don't kill it
			if time.Now().Unix()-10 < workerCmd.created.Unix() {
				log.Debug("killAwolWorkers> Avoid killing baby worker %s born at %s", name, workerCmd.created)
				continue
			}
			log.Info("Killing AWOL worker %s", name)
			kill = true
		} else if w.Status == sdk.StatusDisabled {
			log.Info("Killing disabled worker %s", w.Name)
			kill = true
		}

		if kill {
			if err := h.killWorker(name, workerCmd); err != nil {
				log.Warning("Error killing worker %s :%s", name, err)
			}
			killedWorkers = append(killedWorkers, name)
		}
	}

	for _, name := range killedWorkers {
		delete(h.workers, name)
	}

	return nil
}

// checkRequirement checks binary requirement in path
func (h *HatcheryLocal) checkRequirement(r sdk.Requirement) (bool, error) {
	switch r.Type {
	case sdk.BinaryRequirement:
		if _, err := exec.LookPath(r.Value); err != nil {
			log.Debug("checkRequirement> %v not in path", r.Value)
			// Return nil because the error contains 'Exit status X', that's what we wanted
			return false, nil
		}
		return true, nil
	case sdk.PluginRequirement:
		return true, nil
	case sdk.OSArchRequirement:
		osarch := strings.Split(r.Value, "/")
		if len(osarch) != 2 {
			return false, fmt.Errorf("invalid requirement %s", r.Value)
		}
		return osarch[0] == strings.ToLower(sdk.GOOS) && osarch[1] == strings.ToLower(sdk.GOARCH), nil
	case sdk.NetworkAccessRequirement:
		log.Debug("checkRequirement> checking network requirement:%v", r.Value)
		conn, err := net.DialTimeout("tcp", r.Value, 10*time.Second)
		if err != nil {
			log.Debug("checkRequirement> dial err:%v", err)
			return false, nil
		}
		conn.Close()
		return true, nil
	default:
		log.Debug("checkRequirement> %v don't work on this hatchery", r.Type)
		return false, nil
	}
}
