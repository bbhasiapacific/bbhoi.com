package store

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"bbhoi.com/debug"
	"bbhoi.com/httputil"
	"bbhoi.com/response"
	"bbhoi.com/session"
)

const (
	createUserSQL = `
	id serial PRIMARY KEY,
	email text NOT NULL,
	password text NOT NULL,
	fullname text NOT NULL,
	title text NOT NULL,
	description text NOT NULL,
	avatar_url text NOT NULL,
	interests text[],
	verification_code text NOT NULL,
	is_admin boolean NOT NULL,
	updated_at timestamp NOT NULL,
	created_at timestamp NOT NULL`
)

const (
	UserAvatarURL = `oi-content/user/%d/image`
)

type User interface {
	ID() int64
	Exists() bool
	Update(w http.ResponseWriter, r *http.Request)
	UpdateInterests(w http.ResponseWriter, r *http.Request)
	UpdateAvatar(w http.ResponseWriter, r *http.Request)

	CreateProject(w http.ResponseWriter, r *http.Request)
	UpdateProject(w http.ResponseWriter, r *http.Request)
	DeleteProject(w http.ResponseWriter, r *http.Request)
	JoinProject(w http.ResponseWriter, r *http.Request)
	
	CreateTask(w http.ResponseWriter, r *http.Request)
	UpdateTask(w http.ResponseWriter, r *http.Request)
	DeleteTask(w http.ResponseWriter, r *http.Request)
	ToggleTaskStatus(w http.ResponseWriter, r *http.Request)

	AssignWorker(w http.ResponseWriter, r *http.Request)
	UnassignWorker(w http.ResponseWriter, r *http.Request)

	IsAuthor(projectID int64) bool
	IsAdmin() bool
}

type user struct {
	ID_              int64     `json:"id"`
	IDStr            string    `json:"idStr"`
	Email            string    `json:"email"`
	Password         string    `json:"-"`
	Fullname         string    `json:"fullname"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	AvatarURL        string    `json:"avatarURL"`
	Interests        []byte    `json:"interests"`
	VerificationCode string    `json:"-"`
	IsAdmin_         bool      `json:"isAdmin"`
	UpdatedAt        time.Time `json:"updatedAt"`
	CreatedAt        time.Time `json:"createdAt"`

	CreatedProjectsCount_   int64 `json:"created_projects_count"`
	CompletedProjectsCount_ int64 `json:"completed_projects_count"`
	InvolvedProjectsCount_  int64 `json:"involved_projects_count"`
	CompletedTasksCount_    int64 `json:"completed_tasks_count"`

	MaxCreatedProjectsCount_   int64 `json:"max_created_projects_count"`
	MaxCompletedProjectsCount_ int64 `json:"max_completed_projects_count"`
	MaxInvolvedProjectsCount_  int64 `json:"max_involved_projects_count"`
	MaxCompletedTasksCount_    int64 `json:"max_completed_tasks_count"`
}

func (u user) ID() int64 {
	return u.ID_
}

func (u user) IsAdmin() bool {
	return u.IsAdmin_
}

func (u user) Exists() bool {
	return u.Email != ""
}

func GetUser(userID int64) (User, error) {
	const rawSQL = `
	SELECT * FROM user_ WHERE id = $1 LIMIT 1`

	var u user
	if err := db.QueryRow(rawSQL, userID).Scan(
			&u.ID_,
			&u.Email,
			&u.Password,
			&u.Fullname,
			&u.Title,
			&u.Description,
			&u.AvatarURL,
			&u.Interests,
			&u.VerificationCode,
			&u.IsAdmin_,
			&u.UpdatedAt,
			&u.CreatedAt,
	); err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	u.IDStr = strconv.FormatInt(u.ID_, 10)

	if len(u.AvatarURL) == 0 {
		u.AvatarURL = "avatar.jpg"
	}

	return u, nil
}

func queryUsers(q string, data ...interface{}) ([]User, error) {
	rows, err := db.Query(q, data...)
	if err != nil {
		return nil, debug.Error(err)
	}
	defer rows.Close()

	var us []User
	for rows.Next() {
		var u user

		if err = rows.Scan(
			&u.ID_,
			&u.Email,
			&u.Password,
			&u.Fullname,
			&u.Title,
			&u.Description,
			&u.AvatarURL,
			&u.Interests,
			&u.VerificationCode,
			&u.IsAdmin_,
			&u.UpdatedAt,
			&u.CreatedAt,
		); err != nil {
			return nil, debug.Error(err)
		}

		u.IDStr = strconv.FormatInt(u.ID_, 10)

		if len(u.AvatarURL) == 0 {
			u.AvatarURL = "avatar.jpg"
		}

		us = append(us, u)
	}

	return us, nil
}

func (u user) Update(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		response.ServerError(w, err)
		return
	}

	for k, v := range r.Form {
		switch k {
		case "fullname":
			err = u.updateFullname(v[0])
		case "title":
			err = u.updateTitle(v[0])
		case "description":
			err = u.updateDescription(v[0])
		}

		if err != nil {
			response.ServerError(w, err)
			return
		}
	}

	response.OK(w, nil)
}

func (u user) updateFullname(fullname string) error {
	const rawSQL = `
	UPDATE user_ SET fullname = $1, updated_at = now() WHERE id = $2`

	if _, err := db.Exec(rawSQL, fullname, u.ID_); err != nil {
		return debug.Error(err)
	}

	return nil
}

func (u user) updateTitle(title string) error {
	const rawSQL = `
	UPDATE user_ SET title = $1, updated_at = now() WHERE id = $2`

	if _, err := db.Exec(rawSQL, title, u.ID_); err != nil {
		return debug.Error(err)
	}

	return nil
}

func (u user) updateDescription(description string) error {
	const rawSQL = `
	UPDATE user_ SET description = $1, updated_at = now() WHERE id = $2`

	if _, err := db.Exec(rawSQL, description, u.ID_); err != nil {
		return debug.Error(err)
	}

	return nil
}

func (u user) UpdateInterests(w http.ResponseWriter, r *http.Request) {
	const q = `UPDATE user_ SET interests = $1, updated_at = now() WHERE id = $2`

	interests := strings.Split(r.FormValue("interests"), ",")

	if _, err := db.Exec(q, interests, u.ID_); err != nil {
		response.ServerError(w, err)
		return
	}

	response.OK(w, nil)
}

func (u user) updateAvatarURL(url string) error {
	const q = `UPDATE user_ SET avatar_url = $1, updated_at = now() WHERE id = $2`

	if _, err := db.Exec(q, url, u.ID_); err != nil {
		return debug.Error(err)
	}
	return nil
}

func CreatedProjects(w http.ResponseWriter, r *http.Request) {
	var parser Parser

	userID := parser.Int(r.FormValue("userID"))
	if parser.Err != nil {
		response.ClientError(w, http.StatusBadRequest)
		return
	}

	const rawSQL = `SELECT * FROM project WHERE authorID = $1`

	ps, err := queryProjects(rawSQL, userID)
	if err != nil {
		response.ServerError(w, err)
		return
	}

	response.OK(w, ps)
}

func InvolvedProjects(w http.ResponseWriter, r *http.Request) {
	var parser Parser

	userID := parser.Int(r.FormValue("userID"))
	if parser.Err != nil {
		response.ClientError(w, http.StatusBadRequest)
		return
	}

	const rawSQL = `
	SELECT project.* FROM project
	INNER JOIN member ON member.project_id = project.id
	WHERE member.user_id = $1`

	ps, err := queryProjects(rawSQL, userID)
	if err != nil {
		response.ServerError(w, err)
		return
	}

	response.OK(w, ps)
}

func CompletedProjects(w http.ResponseWriter, r *http.Request) {
	var parser Parser

	userID := parser.Int(r.FormValue("userID"))
	if parser.Err != nil {
		response.ClientError(w, http.StatusBadRequest)
		return
	}

	const rawSQL = `
	SELECT project.* FROM project
	WHERE author_id = $1 AND status = $2`

	ps, err := queryProjects(rawSQL, userID, "completed")
	if err != nil {
		response.ServerError(w, err)
		return
	}

	response.OK(w, ps)
}

func (u user) CreatedProjectsCount() (int64, error) {
	const q = `SELECT COUNT(*) FROM project WHERE authorID = $1`

	return count(q, u.ID_)
}

func (u user) InvolvedProjectsCount() (int64, error) {
	const q = `
	SELECT COUNT(project.*) FROM project
	INNER JOIN member ON member.projectID = project.id
	WHERE member.user_id = $1`

	return count(q, u.ID_)
}

func (u user) CompletedProjectsCount() (int64, error) {
	const q = `
	SELECT COUNT(project.*) FROM project
	WHERE status = 'completed' AND project.authorID = $1`

	return count(q, u.ID_)
}

func MaxCreatedProjectsCount() (int64, error) {
	const q = `
	SELECT MAX(n) FROM (
		SELECT COUNT(*) AS n FROM project
		GROUP BY project.authorID
	) as n`

	return count(q)
}

func MaxInvolvedProjectsCount() (int64, error) {
	const q = `
	SELECT MAX(n) FROM (
		SELECT COUNT(project.*) AS n FROM project
		INNER JOIN member ON member.projectID = project.id
		GROUP BY member.user_id
	) as n`

	return count(q)
}

func MaxCompletedProjectsCount() (int64, error) {
	const q = `
	SELECT MAX(n) FROM (
		SELECT COUNT(project.*) AS n FROM project
		WHERE status = 'completed
		GROUP BY project.authorID'
	) as n`

	return count(q)
}

func (u user) UpdateAvatar(w http.ResponseWriter, r *http.Request) {
	// FIXME: there must be some other way changing directory
	if err := os.Chdir(ContentFolder); err != nil {
		response.ServerError(w, err)
		return
	}

	url := fmt.Sprintf(UserAvatarURL, u.ID_)
	finalURL, header, err := httputil.SaveFileWithExtension(w, r, "image", url)
	if err != nil || header == nil {
		response.ClientError(w, http.StatusBadRequest)
		return
	}

	if err := os.Chdir(".."); err != nil {
		response.ServerError(w, err)
		return
	}

	if err = u.updateAvatarURL(finalURL); err != nil {
		response.ServerError(w, err)
		return
	}

	response.OK(w, nil)
}

func (u user) CreateProject(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	if title == "" {
		response.ClientError(w, http.StatusBadRequest)
		return
	}

	tagline := r.FormValue("tagline")
	description := r.FormValue("description")

	// basic project info
	projectID, err := insertProject(map[string]string{
			"authorID": u.IDStr,
			"title": title,
			"tagline": tagline,
			"description": description,
	})

	if err != nil {
		response.ServerError(w, err)
		return
	}

	var ok bool

	// add author to project user list
	if err = AddMember(projectID, u.ID_); err != nil {
		goto error
	}

	// image
	if ok, err = saveProjectImage(w, r, projectID); err != nil || !ok {
		goto error
	}

	response.OK(w, projectID)
	return

error:
	if err := deleteProject(projectID); err != nil {
		debug.Warn(err)
	}
	response.ServerError(w, err)
}

func (u user) UpdateProject(w http.ResponseWriter, r *http.Request) {
	var parser Parser
	var err error

	projectID := parser.Int(r.FormValue("projectID"))
	if parser.Err != nil {
		response.ClientError(w, http.StatusBadRequest)
		return
	}

	if !u.IsAuthor(projectID) {
		response.ClientError(w, http.StatusForbidden)
		return
	}

	for k, v := range r.Form {
		if len(v) == 0 {
			continue
		}

		switch k {
		case "title":
			err = updateProjectTitle(projectID, v[0])
		case "tagline":
			err = updateProjectTagline(projectID, v[0])
		case "description":
			err = updateProjectDescription(projectID, v[0])
		}
		if err != nil {
			response.ServerError(w, err)
			return
		}
	}

	response.OK(w, nil)
}

func (u user) DeleteProject(w http.ResponseWriter, r *http.Request) {
	var parser Parser

	projectID := parser.Int(r.FormValue("projectID"))
	if parser.Err != nil {
		response.ClientError(w, http.StatusBadRequest)
		return
	}

	if !isAuthor(projectID, u.ID_) {
		response.ClientError(w, http.StatusUnauthorized)
		return
	}

	if err := deleteProject(projectID); err != nil {
		response.ServerError(w, err)
		return
	}
}

func deleteProject(projectID int64) error {
	const rawSQL = `DELETE FROM project WHERE id = $1`

	// delete project
	if _, err := db.Exec(rawSQL, projectID); err != nil {
		return debug.Error(err)
	}

	const rawSQL2 = `DELETE FROM member WHERE project_id = $1`

	// delete project users
	if _, err := db.Exec(rawSQL2, projectID); err != nil {
		return debug.Error(err)
	}

	// delete project image
	if err := os.RemoveAll(fmt.Sprintf("oi-content/project/%d", projectID)); err != nil {
		return debug.Error(err)
	}

	return nil
}

func (u user) JoinProject(w http.ResponseWriter, r *http.Request) {
	var parser Parser

	projectID := parser.Int(r.FormValue("parser"))
	if parser.Err != nil {
		response.ClientError(w, http.StatusBadRequest)
		return
	}

	const rawSQL = `
	INSERT INTO members (projectID, userID, status)
	VALUES ($1, $2, $3)
	WHERE projectID = $4`

	if _, err := db.Exec(rawSQL, projectID, u.ID, "pending"); err != nil {
		response.ServerError(w, err)
	}
	response.OK(w, nil)
}

func (u user) IsAuthor(projectID int64) bool {
	return isAuthor(projectID, u.ID_)
}

func (u user) CreateTask(w http.ResponseWriter, r *http.Request) {
	var taskID int64
	var parser Parser
	var err error

	projectID := parser.Int(r.FormValue("projectID"))
	startDate := parser.Time(r.FormValue("startDate"))
	endDate := parser.Time(r.FormValue("endDate"))
	if parser.Err != nil {
		response.ClientError(w, http.StatusBadRequest)
		return
	}

	if startDate.After(endDate) {
		response.ClientError(w, http.StatusBadRequest)
		return
	}

	if taskID, err = insertTask(insertTaskParams{
		authorID: u.ID_,
		projectID: projectID,
		title: r.FormValue("title"),
		description: r.FormValue("description"),
		done: false,
		tags: r.FormValue("tags"),
		startDate: startDate,
		endDate: endDate,
	}); err != nil {
		response.ServerError(w, err)
		return
	}

	response.OK(w, taskID)
}

func (u user) UpdateTask(w http.ResponseWriter, r *http.Request) {
	var taskID int64
	var err error

	if err = updateTask(updateTaskParams{
		taskID: r.FormValue("taskID"),
		title: r.FormValue("title"),
		description: r.FormValue("description"),
		tags: r.FormValue("tags"),
		startDate: r.FormValue("startDate"),
		endDate: r.FormValue("endDate"),
	}); err != nil {
		response.ServerError(w, err)
		return
	}

	response.OK(w, taskID)
}

func (u user) ToggleTaskStatus(w http.ResponseWriter, r *http.Request) {
	var parser Parser

	taskID := parser.Int(r.FormValue("taskID"))
	if parser.Err != nil {
		response.ClientError(w, http.StatusBadRequest)
		return
	}

	if err := toggleTaskStatus(taskID); err != nil {
		response.ServerError(w, err)
		return
	}

	response.OK(w, taskID)
}

func (u user) DeleteTask(w http.ResponseWriter, r *http.Request) {
	var parser Parser

	projectID := parser.Int(r.FormValue("projectID"))
	if parser.Err != nil {
		response.ClientError(w, http.StatusBadRequest)
		return
	}

	// check if user is member of the project
	if !IsMember(projectID, u.ID_) {
		response.ClientError(w, http.StatusForbidden)
		return
	}

	if err := deleteTask(deleteTaskParams{
		taskID: r.FormValue("taskID"),
	}); err != nil {
		response.ServerError(w, err)
		return
	}

	response.OK(w, nil)
}

func (u user) AssignWorker(w http.ResponseWriter, r *http.Request) {
	var parser Parser

	taskID := parser.Int(r.FormValue("taskID"))
	userID := parser.Int(r.FormValue("userID"))
	if parser.Err != nil {
		response.ClientError(w, http.StatusBadRequest)
		return
	}

	if r.FormValue("toggle") == "true" {
		if err := toggleWorker(taskID, userID, u.ID_); err != nil {
			response.ServerError(w, err)
		}
		response.OK(w, taskID)
		return
	}

	insertWorker(taskID, userID, u.ID_)
}

func (u user) UnassignWorker(w http.ResponseWriter, r *http.Request) {
	var parser Parser

	taskID := parser.Int(r.FormValue("taskID"))
	userID := parser.Int(r.FormValue("userID"))
	if parser.Err != nil {
		response.ClientError(w, http.StatusBadRequest)
		return
	}

	deleteWorker(taskID, userID)
}

func SetAdmin(w http.ResponseWriter, r *http.Request) {
	var parser Parser

	const rawSQL = `
	UPDATE user_ WHERE id = $1 WHERE is_admin = false
	SET is_admin = true`

	userID := parser.Int(r.FormValue("userID"))
	if parser.Err != nil {
		response.ClientError(w, http.StatusBadRequest)
		return
	}

	if _, err := db.Exec(rawSQL, userID); err != nil {
		response.ServerError(w, err)
	}
}

func UnsetAdmin(w http.ResponseWriter, r *http.Request) {
	var parser Parser

	const rawSQL = `
	UPDATE user_ WHERE id = $1 WHERE is_admin = true
	SET is_admin = false`

	id := parser.Int(r.FormValue("id"))
	if parser.Err != nil {
		response.ClientError(w, http.StatusBadRequest)
		return
	}

	if _, err := db.Exec(rawSQL, id); err != nil {
		response.ServerError(w, err)
	}
}

func GetAdmins(w http.ResponseWriter, r *http.Request) {
	var parser Parser

	count := parser.Int(r.FormValue("count"))
	if parser.Err != nil {
		response.ClientError(w, http.StatusBadRequest)
		return
	}

	const rawSQL = `
	SELECT * FROM user_ WHERE is_admin = true
	LIMIT $1`

	users, err := queryUsers(rawSQL, count)
	if err != nil {
		response.ServerError(w, err)
		return
	}

	response.OK(w, users)
}

func GetUserByID(userID int64) (User, error) {
	const rawSQL = `
	SELECT * FROM user_ WHERE id = $1`

	var u user
	if err := db.QueryRow(rawSQL, userID).Scan(
		&u.ID_,
		&u.Email,
		&u.Password,
		&u.Fullname,
		&u.Title,
		&u.Description,
		&u.AvatarURL,
		&u.Interests,
		&u.VerificationCode,
		&u.IsAdmin_,
		&u.UpdatedAt,
		&u.CreatedAt,
	); err != nil && err != sql.ErrNoRows {
		return nil, debug.Error(err)
	}

	u.IDStr = strconv.FormatInt(u.ID_, 10)

	if len(u.AvatarURL) == 0 {
		u.AvatarURL = "avatar.jpg"
	}

	return u, nil
}

func GetUserByEmail(email string) (User, error) {
	const rawSQL = `
	SELECT * FROM user_ WHERE email = $1`

	var u user
	if err := db.QueryRow(rawSQL, email).Scan(
		&u.ID_,
		&u.Email,
		&u.Password,
		&u.Fullname,
		&u.Title,
		&u.Description,
		&u.AvatarURL,
		&u.Interests,
		&u.VerificationCode,
		&u.IsAdmin_,
		&u.UpdatedAt,
		&u.CreatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, debug.Error(err)
	}

	u.IDStr = strconv.FormatInt(u.ID_, 10)

	if len(u.AvatarURL) == 0 {
		u.AvatarURL = "avatar.jpg"
	}

	return u, nil
}

func CurrentUser(r *http.Request) User {
	user, err := GetUserByEmail(session.GetEmail(r))
	if err != nil {
		debug.Warn(err)
		return nil
	}
	return user
}
