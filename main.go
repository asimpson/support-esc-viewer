package main

import (
	"context"
	_ "embed"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

//go:embed template.html
var htmlTemplate string

// Notification is a struct that stores information about a notification
type Notification struct {
	Subject    github.NotificationSubject
	ID         string
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
}

type Notifications struct {
	Notifications []Notif
	RefreshedTime string
	IssueCount    int
	Time          string
}

type Notif struct {
	Body     string
	Title    string
	URL      string
	Id       string
	Labels   []github.Label
	Comments []Comment
	Closed   bool
}

type Comment struct {
	Body  string
	Title string
	URL   string
	Date  string
}

// NotificationList is a list of notifications
type NotificationList []Notification

var client *github.Client
var ctx = context.Background()

func getNotif(ts oauth2.TokenSource) (data Notifications) {
	//store the current time in a variable in the format YYYY-MM-DDTHH:MM:SSZ
	//this is used to compare the time of the notification with the current time
	//to determine if the notification is new
	now := time.Now().Format(time.RFC3339)

	tc := oauth2.NewClient(ctx, ts)
	client = github.NewClient(tc)

	//get every page of notifications from the github api
	//and store them in a list
	var notifications NotificationList
	var notifcationData []Notif

	opt := &github.NotificationListOptions{ListOptions: github.ListOptions{PerPage: 50}}
	for {
		notifs, resp, err := client.Activity.ListNotifications(ctx, opt)
		if err != nil {
			log.Fatal(err)
		}
		for _, n := range notifs {
			if strings.Contains(*n.Repository.FullName, "support-escalations") {
				notifications = append(notifications, Notification{
					Subject: *n.Subject,
					ID:      *n.ID,
					Repository: struct {
						FullName string "json:\"full_name\""
					}{
						FullName: *n.Repository.FullName,
					},
				})
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	//loop through all notifications in the notification list
	for _, notification := range notifications {
		//split the url to get the owner and repo
		ownerAndRepo := strings.Split(notification.Repository.FullName, "/")
		owner := ownerAndRepo[0]
		repo := ownerAndRepo[1]
		//split the url to get the issue number
		url := strings.Split(*notification.Subject.URL, "/")
		number, err := strconv.Atoi(url[len(url)-1])
		if err != nil {
			log.Fatal(err)
		}

		//get the issue
		//client.Issues.ListByRepo()
		issue, _, err := client.Issues.Get(ctx, owner, repo, number)
		if err != nil {
			log.Fatal(err)
		}

		if *issue.Body != "" {
			var notif Notif
			notif.Body = string(markdown.ToHTML([]byte(*issue.Body), nil, nil))
			notif.Title = *issue.Title
			notif.URL = *issue.HTMLURL
			notif.Id = notification.ID
			notif.Labels = issue.Labels
			notif.Closed = *issue.State == "closed"

			//if there are issue comments, add them to the card
			if *issue.Comments > 0 {
				opt := &github.IssueListCommentsOptions{Sort: "created", Direction: "desc"}
				comments, _, err := client.Issues.ListComments(ctx, owner, repo, number, opt)
				if err != nil {
					log.Fatal(err)
				}
				for _, c := range comments {
					if *c.Body != "" {
						var comment Comment
						comment.Body = string(markdown.ToHTML([]byte(*c.Body), nil, nil))
						comment.Title = *c.User.Login
						comment.URL = *c.HTMLURL
						comment.Date = time.Since(*c.CreatedAt).Round(time.Hour).String()
						notif.Comments = append(notif.Comments, comment)
					}
				}
			}
			notifcationData = append(notifcationData, notif)
		}
	}

	t, err := time.Parse(time.RFC3339, now)
	if err != nil {
		log.Fatal(err)
	}
	data.Notifications = notifcationData
	data.RefreshedTime = t.Format("15:04")
	data.Time = now
	data.IssueCount = len(notifications)
	return data
}

func main() {
	token := os.Getenv("GITHUB_TOKEN")

	//create a new github client
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	http.HandleFunc("/read/", func(w http.ResponseWriter, r *http.Request) {
		tc := oauth2.NewClient(ctx, ts)
		client = github.NewClient(tc)

		//convert timestamp to time
		t, _ := time.Parse(time.RFC3339, r.URL.Query().Get("time"))
		//mark notifications in the repo support-escalations as read with last_read_at set to the timestamp
		resp, err := client.Activity.MarkRepositoryNotificationsRead(ctx, "grafana", "support-escalations", t)
		if err != nil {
			log.Fatal(err)
		}
		if resp.StatusCode == http.StatusResetContent {
			//redirect to the home page
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		data := getNotif(ts)
		h := template.Must(template.New("template").Parse(htmlTemplate))
		h.Execute(w, data)
	})
	//start the server on port 4000
	log.Fatal(http.ListenAndServe(":4000", nil))
}
