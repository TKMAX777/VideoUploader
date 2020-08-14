package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"../lib"
)

//MyPageHandle handle my page
func MyPageHandle(w http.ResponseWriter, r *http.Request) {
	var user lib.User

	var V MyPage = MyPage{
		Header: Header{
			Title:    "MyPage",
			UserName: os.Getenv("REMOTE_USER"),
			Error:    ErrorHandle(r.URL.Query().Get("Error")),
			Success:  SuccessHandle(r.URL.Query().Get("Success")),
		},
		Slack:  Settings.SlackWebhook != "",
		Footer: Footer{},
	}

	if err := user.Get(os.Getenv("REMOTE_USER")); err != nil {
		V.Video = []lib.Video{}
	} else {
		V.Video = user.Video
	}

	V.Header.User = user

	t, e := template.New("").ParseFiles(
		filepath.Join("resources", "header.html"),
		filepath.Join("resources", "mypage.html"),
		filepath.Join("resources", "style.html"),
		filepath.Join("resources", "footer.html"),
		filepath.Join("resources", "script.html"),
	)

	if e != nil {
		fmt.Fprintf(w, "%s", e.Error())
		return
	}

	if e = t.ExecuteTemplate(w, "mypage", V); e != nil {
		fmt.Fprintf(w, "%s", e.Error())
		return
	}
	return

}

//UpdateVideoInfo update user video infomation
func UpdateVideoInfo(w http.ResponseWriter, r *http.Request) {
	var getQuery GetQuery = make(GetQuery)
	getQuery["Page"] = "MyPage"

	// error handling
	defer func() {
		err := recover()
		if err != nil {
			fmt.Fprintf(w, "Panic: %v\n", err)
		}
	}()

	r.ParseForm()

	var user lib.User
	if e := user.Get(os.Getenv("REMOTE_USER")); e != nil {
		getQuery["Error"] = "NotFound"
		lib.Logger(e)
		w.Header().Set("Location", "index.up"+getQuery.Encode())
		w.WriteHeader(http.StatusTemporaryRedirect)
		return
	}

	var video = lib.SearchVideo(user.Video, r.URL.Query().Get("Video"))
	if video.Video == "" {
		getQuery["Error"] = "NotFound"
		w.Header().Set("Location", "index.up"+getQuery.Encode())
		w.WriteHeader(http.StatusTemporaryRedirect)
		return
	}

	// update thumbnail
	func() {
		thumb, thumbH, err := r.FormFile("thumbnail")
		if err != nil || thumbH.Filename == "" {
			return
		}

		filet, err := os.Create(filepath.Join("tmp", filepath.Base(strings.ReplaceAll(thumbH.Filename, "..", "_"))))
		if err != nil {
			lib.Logger(err)
			return
		}

		if _, err = io.Copy(filet, thumb); err != nil {
			lib.Logger(err)
			return
		}

		filet.Close()

		if err := lib.ImageConverter(thumb, filepath.Join("Videos", video.Video+".png")); err != nil {
			lib.Logger(err)
			return
		}

	}()

	// update title
	func() {
		var title = r.FormValue("title")
		var newTitle string = strings.ReplaceAll(strings.TrimSpace(title), "\n", "")

		if strings.ReplaceAll(strings.TrimSpace(newTitle), "\n", "") == "" {
			return
		}

		video.Title = newTitle
		return
	}()

	var err = video.Update()
	if err != nil {
		lib.Logger(err)
		getQuery["Error"] = "UpdateError"
		w.Header().Set("Location", "index.up"+getQuery.Encode())
		w.WriteHeader(http.StatusTemporaryRedirect)
		return
	}

	getQuery["Success"] = "update"

	w.Header().Set("Location", "index.up"+getQuery.Encode())
	w.WriteHeader(http.StatusTemporaryRedirect)
}
