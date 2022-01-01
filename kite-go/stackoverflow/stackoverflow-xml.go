//go:generate ./xmlgen.py stackoverflow-xml.go
//go:generate protoc --go_out=. types.proto xmlgen.proto

package stackoverflow

import "encoding/xml"

type XMLPost struct {
	XMLName               xml.Name `xml:"row"`
	Id                    int64    `xml:",attr"`
	PostTypeId            int64    `xml:",attr"`
	ParentId              int64    `xml:",attr"`
	AcceptedAnswerId      int64    `xml:",attr"`
	CreationDate          string   `xml:",attr"`
	Score                 int64    `xml:",attr"`
	ViewCount             int64    `xml:",attr"`
	Body                  string   `xml:",attr"`
	OwnerUserId           int64    `xml:",attr"`
	LastEditorUserId      int64    `xml:",attr"`
	LastEditorDisplayName string   `xml:",attr"`
	LastEditDate          string   `xml:",attr"`
	LastActivityDate      string   `xml:",attr"`
	Title                 string   `xml:",attr"`
	Tags                  string   `xml:",attr"`
	AnswerCount           int64    `xml:",attr"`
	CommentCount          int64    `xml:",attr"`
	FavoriteCount         int64    `xml:",attr"`
	CommunityOwnedDate    string   `xml:",attr"`
}

// --

type XMLUser struct {
	XMLName        xml.Name `xml:"row"`
	Id             uint64   `xml:",attr"`
	Reputation     int64    `xml:",attr"`
	CreationDate   string   `xml:",attr"`
	DisplayName    string   `xml:",attr"`
	LastAccessDate string   `xml:",attr"`
	WebsiteUrl     string   `xml:",attr"`
	Location       string   `xml:",attr"`
	AboutMe        string   `xml:",attr"`
	Views          int64    `xml:",attr"`
	UpVotes        int64    `xml:",attr"`
	DownVotes      int64    `xml:",attr"`
	Age            int64    `xml:",attr"`
	AccountId      int64    `xml:",attr"`
}

// --

type XMLVote struct {
	XMLName      xml.Name `xml:"row"`
	Id           int64    `xml:",attr"`
	PostId       int64    `xml:",attr"`
	VoteTypeId   int64    `xml:",attr"`
	CreationDate string   `xml:",attr"`
}

// --

type XMLPostLink struct {
	XMLName       xml.Name `xml:"row"`
	Id            int64    `xml:",attr"`
	CreationDate  string   `xml:",attr"`
	PostId        int64    `xml:",attr"`
	RelatedPostId int64    `xml:",attr"`
	LinkTypeId    int64    `xml:",attr"`
}
