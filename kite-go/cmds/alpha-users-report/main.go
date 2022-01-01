package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-golib/email"
	_ "github.com/lib/pq"
	mixpanel "github.com/noahlt/go-mixpanel"
)

var (
	addr = mail.Address{
		Name:    "Kite Engineering",
		Address: "eng@kite.com",
	}
	hostport = os.Getenv("KITE_ENG_EMAIL_SMTP_HOSTPORT")
	user     = os.Getenv("KITE_ENG_EMAIL_USERNAME")
	password = os.Getenv("KITE_ENG_EMAIL_PASSWORD")
)

func active(u *AlphaUserInfo, d time.Duration, now time.Time) bool {
	return u.CreatedAt.Before(now.Add(-d)) && u.LastFocus.After(now.Add(-d))
}

func main() {
	now := time.Now()
	today := now.Format("2006-01-02")

	var (
		to       string
		start    string
		end      string
		printOut bool
	)
	flag.StringVar(&to, "to", "", "comma-separated list of recipients of the email, or \"stdout\" to print to output")
	flag.StringVar(&start, "start", "20151001", "earliest date from which to get events")
	flag.StringVar(&end, "end", today, "latest date from which to get events")
	flag.BoolVar(&printOut, "print", false, "print to stdout rather than send an email")
	flag.Parse()

	if to == "" && !printOut {
		log.Fatalln("You must specify a recipient using the --to flag or specify --print to print to stdout")
	}
	var recipients []string
	for _, r := range strings.Split(to, ",") {
		recipients = append(recipients, strings.TrimSpace(r))
	}

	mp := mixpanel.NewMixpanelFromEnv()

	params := map[string]string{
		"from_date": "2015-10-01",
		"to_date":   today,
		"event":     "Authenticated,App focused",
	}

	events, err := mp.ExportQuery(params)
	if err != nil {
		log.Fatalln(err)
	}

	var userInfos []*AlphaUserInfo
	var dailyActives []*AlphaUserInfo
	var weeklyActives []*AlphaUserInfo
	var monthlyActives []*AlphaUserInfo

	for _, user := range getAllUsers() {
		userInfo := getInfoForUser(user, events)
		userInfos = append(userInfos, userInfo)

		if active(userInfo, time.Hour*24, now) {
			dailyActives = append(dailyActives, userInfo)
		}

		if active(userInfo, time.Hour*24*7, now) {
			weeklyActives = append(weeklyActives, userInfo)
		}

		if active(userInfo, time.Hour*24*28, now) {
			monthlyActives = append(monthlyActives, userInfo)
		}
	}

	buf := &bytes.Buffer{}
	err = tmpl.Execute(buf, struct {
		UserInfos      []*AlphaUserInfo
		DailyActives   []*AlphaUserInfo
		WeeklyActives  []*AlphaUserInfo
		MonthlyActives []*AlphaUserInfo
		Today          string
	}{
		UserInfos:      userInfos,
		DailyActives:   dailyActives,
		WeeklyActives:  weeklyActives,
		MonthlyActives: monthlyActives,
		Today:          today,
	})
	if err != nil {
		log.Fatalln(err)
	}

	if printOut {
		fmt.Print(buf.String())
	} else {
		message := email.Message{
			To:      recipients,
			Subject: fmt.Sprintf("Alpha Users Report (%s)", today),
			Body:    buf.Bytes(),
			HTML:    true,
		}

		client, err := email.NewClient(hostport, user, password)
		if err != nil {
			log.Fatalln("unable to start email client:", err)
		}

		err = client.Send(addr, message)
		if err != nil {
			log.Fatalln(err)
		}
	}
}

// Get user info

// AlphaUserInfo has all the data for one row of our alpha users table.
type AlphaUserInfo struct {
	ID        int64
	Name      string
	Email     string
	CreatedAt time.Time
	LastFocus time.Time
	Focused   bool
}

func getInfoForUser(user *community.User, events []mixpanel.ExportQueryResult) *AlphaUserInfo {
	var foundFocusEvent bool
	var lastFocus time.Time
	for _, event := range events {
		eventTimeFloat, ok := event.Properties["time"].(float64)
		if !ok {
			fmt.Printf("%+v\n", event)
			fmt.Println("can't get 'time' field from event properties!")
			continue
		}
		eventTime := time.Unix(int64(eventTimeFloat), 0)

		userIDfloat, ok := event.Properties["userId"].(float64)
		if !ok {
			// no userId means the user is not authenticated yet at the time of this event
			continue
		}
		userID := int64(userIDfloat)

		if userID == user.ID &&
			event.Event == "App focused" &&
			(!foundFocusEvent || eventTime.After(lastFocus)) {
			foundFocusEvent = true
			lastFocus = eventTime
		}
	}
	return &AlphaUserInfo{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		LastFocus: lastFocus,
		Focused:   foundFocusEvent,
	}
}

func getAllUsers() []*community.User {
	db := community.DB(os.Getenv("COMMUNITY_DB_DRIVER"), os.Getenv("COMMUNITY_DB_URI"))
	var users []*community.User
	if err := db.Order("id desc").Find(&users).Error; err != nil {
		log.Fatalln(err)
	}
	return users
}

// Render template

var tmpl = template.Must(template.New("report").Parse(`<html>
<head>
<title>Kite Alpha Users</title>
<style type="text/css">
body {
    font-family: 'Helvetica Neue', sans-serif;
}
h1 {
    font-size: 1.2em;
    margin: 1rem;
}
</style>
</head>
<body>
<h1>Kite Alpha Users ({{.Today}})</h1>
<table cellspacing="10">
<tr>
  <th align="left">id</th>
  <th align="left">name</th>
  <th align="left">email</th>
  <th align="left">created at</th>
  <th align="left">last focused app</th>
</tr>
{{range .UserInfos}}
<tr>
  <td>{{.ID}}</td>
  <td>{{.Name}}</td>
  <td>{{.Email}}</td>
  <td>{{.CreatedAt.Format "2006-01-02" }}</td>
  <td>{{if .Focused}}{{.LastFocus.Format "2006-01-02" }}{{else}}-{{end}}</td>
</tr>
{{end}}
</table>

<h1>Daily Active Users: {{len .DailyActives}}</h1>
<table cellspacing="10">
<tr>
  <th align="left">id</th>
  <th align="left">name</th>
  <th align="left">email</th>
  <th align="left">created at</th>
  <th align="left">last focused app</th>
</tr>
{{range .DailyActives}}
<tr>
  <td>{{.ID}}</td>
  <td>{{.Name}}</td>
  <td>{{.Email}}</td>
  <td>{{.CreatedAt.Format "2006-01-02" }}</td>
  <td>{{if .Focused}}{{.LastFocus.Format "2006-01-02" }}{{else}}-{{end}}</td>
</tr>
{{end}}
</table>

<h1>Weekly Active Users: {{len .WeeklyActives}}</h1>
<table cellspacing="10">
<tr>
  <th align="left">id</th>
  <th align="left">name</th>
  <th align="left">email</th>
  <th align="left">created at</th>
  <th align="left">last focused app</th>
</tr>
{{range .WeeklyActives}}
<tr>
  <td>{{.ID}}</td>
  <td>{{.Name}}</td>
  <td>{{.Email}}</td>
  <td>{{.CreatedAt.Format "2006-01-02" }}</td>
  <td>{{if .Focused}}{{.LastFocus.Format "2006-01-02" }}{{else}}-{{end}}</td>
</tr>
{{end}}
</table>

<h1>Monthly Active Users {{len .MonthlyActives}}</h1>
<table cellspacing="10">
<tr>
  <th align="left">id</th>
  <th align="left">name</th>
  <th align="left">email</th>
  <th align="left">created at</th>
  <th align="left">last focused app</th>
</tr>
{{range .MonthlyActives}}
<tr>
  <td>{{.ID}}</td>
  <td>{{.Name}}</td>
  <td>{{.Email}}</td>
  <td>{{.CreatedAt.Format "2006-01-02" }}</td>
  <td>{{if .Focused}}{{.LastFocus.Format "2006-01-02" }}{{else}}-{{end}}</td>
</tr>
{{end}}
</table>

<body>
</html>
`))
