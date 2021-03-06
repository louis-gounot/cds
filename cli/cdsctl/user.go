package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

var userCmd = cli.Command{
	Name:  "user",
	Short: "Manage CDS user",
}

func usr() *cobra.Command {
	return cli.NewCommand(userCmd, nil, []*cobra.Command{
		cli.NewGetCommand(userMeCmd, userMeRun, nil),
		cli.NewListCommand(userListCmd, userListRun, nil),
		cli.NewGetCommand(userShowCmd, userShowRun, nil),
		cli.NewCommand(userResetCmd, userResetRun, nil),
		cli.NewCommand(userConfirmCmd, userConfirmRun, nil),
		cli.NewCommand(userFavoriteCmd, userFavoriteRun, nil),
	})
}

var userListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS users",
}

func userListRun(v cli.Values) (cli.ListResult, error) {
	users, err := client.UserList()
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(users), nil
}

var userMeCmd = cli.Command{
	Name:  "me",
	Short: "Show Current CDS user details",
}

func userMeRun(v cli.Values) (interface{}, error) {
	u, err := client.UserGet(cfg.User)
	if err != nil {
		return nil, err
	}
	var res = struct {
		URL      string `cli:"url"`
		Username string `cli:"username,key"`
		Fullname string `cli:"fullname"`
		Email    string `cli:"email"`
	}{
		URL:      cfg.Host,
		Username: u.Username,
		Fullname: u.Fullname,
		Email:    u.Email,
	}
	return res, nil
}

var userShowCmd = cli.Command{
	Name:  "show",
	Short: "Show CDS user details",
	Args: []cli.Arg{
		{Name: "username"},
	},
}

func userShowRun(v cli.Values) (interface{}, error) {
	u, err := client.UserGet(v.GetString("username"))
	if err != nil {
		return nil, err
	}
	return *u, nil
}

var userResetCmd = cli.Command{
	Name:  "reset",
	Short: "Reset CDS user password",
	OptionalArgs: []cli.Arg{
		{Name: "username"},
		{Name: "email"},
	},
}

func userResetRun(v cli.Values) error {
	username := v.GetString("username")
	if username == "" {
		username = cfg.User
	}
	if username == "" {
		fmt.Printf("Username: ")
		username = cli.ReadLine()
	} else {
		fmt.Println("Username:", username)
	}

	email := v.GetString("email")
	if email == "" {
		fmt.Printf("Email: ")
		email = cli.ReadLine()
	} else {
		fmt.Println("Email:", email)
	}

	if err := client.UserReset(username, email, "cdsctl user confirm %s %s"); err != nil {
		return err
	}
	fmt.Println("Reset done, please check your emails")
	return nil
}

var userConfirmCmd = cli.Command{
	Name:  "confirm",
	Short: "Confirm CDS user password",
	Args: []cli.Arg{
		{Name: "username"},
		{Name: "token"},
	},
	Flags: []cli.Flag{
		{
			Name:      "api-url",
			ShortHand: "H",
			Usage:     "CDS API URL",
			IsValid: func(s string) bool {
				match, _ := regexp.MatchString(`http[s]?:\/\/(.*)`, s)
				return match
			},
		}, {
			Name:  "env",
			Usage: "Display the commands to set up the environment for the cds client",
			Type:  cli.FlagBool,
		},
	},
}

func userConfirmRun(v cli.Values) error {
	var apiURL = v.GetString("api-url")
	if strings.HasSuffix(apiURL, "/") {
		fmt.Fprintf(os.Stderr, "Invalid URL. Remove trailing '/'\n")
	}

	var username = v.GetString("username")
	var token = v.GetString("token")
	var env = v.GetBool("env")
	var insecureSkipVerifyTLS = v.GetBool("insecure")

	conf := cdsclient.Config{
		Host:    apiURL,
		Verbose: os.Getenv("CDS_VERBOSE") == "true",
	}

	client = cdsclient.New(conf)

	ok, password, err := client.UserConfirm(username, token)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("verification failed")
	}

	return doLogin(apiURL, username, password, env, insecureSkipVerifyTLS)
}

var userFavoriteCmd = cli.Command{
	Name:  "favorite",
	Short: "Display all the user favorites",
}

func userFavoriteRun(v cli.Values) error {
	urlUI, err := client.ConfigUser()
	if err != nil {
		return nil
	}
	uiURL := urlUI[sdk.ConfigURLUIKey]

	navbarInfos, err := client.Navbar()
	if err != nil {
		return err
	}

	projFavs := []sdk.NavbarProjectData{}
	wfFavs := []sdk.NavbarProjectData{}
	for _, elt := range navbarInfos {
		if elt.Favorite {
			switch elt.Type {
			case "workflow":
				wfFavs = append(wfFavs, elt)
			case "project":
				projFavs = append(projFavs, elt)
			}
		}
	}

	fmt.Println(" -=-=-=-=- Projects bookmarked -=-=-=-=-")
	for _, prj := range projFavs {
		fmt.Printf("- %s %s\n", prj.Name, uiURL+"/project/"+prj.Key)
	}

	fmt.Println("\n -=-=-=-=- Workflows bookmarked -=-=-=-=-")
	for _, wf := range wfFavs {
		fmt.Printf("- %s %s\n", wf.WorkflowName, uiURL+"/project/"+wf.Key+"/workflow/"+wf.WorkflowName)
	}

	return nil
}
