package stackoverflow

// From http://meta.stackexchange.com/questions/2677/database-schema-documentation-for-the-public-data-dump-and-sede

// PostLinkType is an enum specifying the type of PostLink
type PostLinkType int64

// Values for various PostLinkTypes
const (
	PostLinkTypeLinked    = 1
	PostLinkTypeDuplicate = 3
)

// PostType is an enum specifying the type of Post
type PostType int64

// Values for various PostTypes
const (
	PostTypeQuestion            = 1
	PostTypeAnswer              = 2
	PostTypeOrphanedTagWiki     = 3
	PostTypeTagWikiExcerpt      = 4
	PostTypeTagWiki             = 5
	PostTypeModeratorNomination = 6
	PostTypeWikiPlaceholder     = 7
	PostTypePrivilegeWiki       = 8
)

// VoteType is an enum specifying the type of Vote
type VoteType int64

// Values for various VoteTypes
const (
	VoteTypeAcceptedByOriginator  = 1
	VoteTypeUpMod                 = 2
	VoteTypeDownMod               = 3
	VoteTypeOffensive             = 4
	VoteTypeFavorite              = 5
	VoteTypeClose                 = 6
	VoteTypeReopen                = 7
	VoteTypeBountyStart           = 8
	VoteTypeBountyClose           = 9
	VoteTypeDeletion              = 10
	VoteTypeUndeletion            = 11
	VoteTypeSpam                  = 12
	VoteTypeModeratorReview       = 15
	VoteTypeApproveEditSuggestion = 16
)
