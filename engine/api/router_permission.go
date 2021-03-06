package api

import (
	"fmt"
	"strconv"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func loadGroupPermissionInUser(db gorp.SqlExecutor, groupID int64, u *sdk.User) error {
	permProj, err := project.LoadPermissions(db, groupID)
	if err != nil {
		return sdk.WrapError(err, "Unable to load project permissions for %s", u.Username)
	}
	if u.Permissions.ProjectsPerm == nil {
		u.Permissions.ProjectsPerm = make(map[string]int, len(permProj))
	}
	for _, p := range permProj {
		if u.Permissions.ProjectsPerm[p.Project.Key] < p.Permission {
			u.Permissions.ProjectsPerm[p.Project.Key] = p.Permission
		}
	}

	permWorkflow, err := workflow.LoadWorkflowByGroup(db, groupID)
	if err != nil {
		return sdk.WrapError(err, "Unable to load workflow permissions for  %s", u.Username)
	}
	if u.Permissions.WorkflowsPerm == nil {
		u.Permissions.WorkflowsPerm = make(map[string]int, len(permWorkflow))
	}
	for _, p := range permWorkflow {
		k := sdk.UserPermissionKey(p.Workflow.ProjectKey, p.Workflow.Name)
		if u.Permissions.WorkflowsPerm[k] < p.Permission {
			u.Permissions.WorkflowsPerm[k] = p.Permission
		}
	}
	return nil
}

// loadUserPermissions retrieves all group memberships
func loadUserPermissions(db gorp.SqlExecutor, store cache.Store, u *sdk.User) error {
	u.Groups = nil
	kp := cache.Key("users", u.Username, "perms")
	kg := cache.Key("users", u.Username, "groups")
	okp, errp := store.Get(kp, &u.Permissions)
	if errp != nil {
		log.Error("cannot get from cache %s: %v", kp, errp)
	}
	okg, errg := store.Get(kg, &u.Groups)
	if errg != nil {
		log.Error("cannot get from cache %s: %v", kg, errg)
	}
	if !okp || !okg {
		query := `
			SELECT "group".id, "group".name, "group_user".group_admin
			FROM "group"
	 		JOIN group_user ON group_user.group_id = "group".id
	 		WHERE group_user.user_id = $1 ORDER BY "group".name ASC`

		rows, err := db.Query(query, u.ID)
		if err != nil {
			return sdk.WrapError(err, "Unable to load user groups %s", u.Username)
		}
		defer rows.Close()

		for rows.Next() {
			var group sdk.Group
			var admin bool
			if err := rows.Scan(&group.ID, &group.Name, &admin); err != nil {
				return sdk.WrapError(err, "Unable scan groups %s", u.Username)
			}
			u.Permissions.Groups = append(u.Permissions.Groups, group.Name)
			if admin {
				u.Permissions.Groups = append(u.Permissions.GroupsAdmin, group.Name)
				usr := *u
				usr.Groups = nil
				group.Admins = append(group.Admins, usr)
			}
			if err := loadGroupPermissionInUser(db, group.ID, u); err != nil {
				return err
			}
			u.Groups = append(u.Groups, group)
		}

		if err := store.SetWithTTL(kp, u.Permissions, 120); err != nil {
			log.Error("cannot SetWithTTL kp: %s: %v", kp, err)
		}
		if err := store.SetWithTTL(kg, u.Groups, 120); err != nil {
			log.Error("cannot SetWithTTL kg: %s: %v", kg, err)
		}

	}
	return nil
}

// loadGroupPermissions retrieves all group memberships
func loadPermissionsByGroupID(db gorp.SqlExecutor, store cache.Store, groupID int64) (sdk.Group, sdk.UserPermissions, error) {
	u := sdk.User{}
	g := sdk.Group{
		ID: groupID,
	}
	kg := cache.Key("groups", strconv.Itoa(int(groupID)))
	ku := cache.Key("groups", strconv.Itoa(int(groupID)), "perms")
	find, err := store.Get(kg, &g)
	if err != nil {
		log.Error("cannot get from cache %s: %v", kg, err)
	}
	if !find {
		query := `SELECT "group".name FROM "group" WHERE "group".id = $1`
		if err := db.QueryRow(query, groupID).Scan(&g.Name); err != nil {
			return g, sdk.UserPermissions{}, fmt.Errorf("no group with id %d: %s", groupID, err)
		}
		if err := store.SetWithTTL(kg, g, 120); err != nil {
			log.Error("cannot SetWithTTL: %s: %v", kg, err)
		}
	}

	find2, err2 := store.Get(ku, &u.Permissions)
	if err2 != nil {
		log.Error("cannot get from cache %s: %v", ku, err2)
	}
	if !find2 {
		if err := loadGroupPermissionInUser(db, groupID, &u); err != nil {
			return g, sdk.UserPermissions{}, sdk.WrapError(err, "loadPermissionsByGroupID")
		}
		if err := store.SetWithTTL(ku, u.Permissions, 120); err != nil {
			log.Error("cannot SetWithTTL: %s: %v", ku, err)
		}
	}

	return g, u.Permissions.Clone(), nil
}
