module github.com/kiteco/kiteco

go 1.15

require (
	github.com/Azure/azure-sdk-for-go v48.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.10
	github.com/Azure/go-autorest/autorest/adal v0.9.5
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Azure/go-autorest/autorest/validation v0.3.0 // indirect
	github.com/Microsoft/go-winio v0.4.15 // indirect
	github.com/PuerkitoBio/goquery v1.6.0
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d
	github.com/alecthomas/chroma v0.6.9
	github.com/alexflint/go-arg v1.3.0
	github.com/asaskevich/govalidator v0.0.0-20200907205600-7a23bdc65eef
	github.com/aws/aws-sdk-go v1.37.14
	github.com/biogo/cluster v0.0.0-20150317015930-38d63f016191
	github.com/blend/go-sdk v1.1.1 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/codegangsta/negroni v1.0.0
	github.com/customerio/go-customerio v1.1.0
	github.com/denisenkom/go-mssqldb v0.0.0-20191124224453-732737034ffd // indirect
	github.com/dgraph-io/ristretto v0.0.3
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/dgryski/go-spooky v0.0.0-20170606183049-ed3d087f40e2
	github.com/distatus/battery v0.10.0
	github.com/djimenez/iconv-go v0.0.0-20160305225143-8960e66bd3da
	github.com/dlclark/regexp2 v1.2.0 // indirect
	github.com/dnaeon/go-vcr v1.1.0 // indirect
	github.com/docker/docker v17.12.0-ce-rc1.0.20200505174321-1655290016ac+incompatible
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/dukex/mixpanel v0.0.0-20180925151559-f8d5594f958e
	github.com/dustin/go-humanize v1.0.0
	github.com/elazarl/go-bindata-assetfs v1.0.1
	github.com/erikstmartin/go-testdb v0.0.0-20160219214506-8d10e4a1bae5 // indirect
	github.com/fluent/fluent-logger-golang v1.5.0
	github.com/frankban/quicktest v1.11.2 // indirect
	github.com/fsnotify/fsnotify v1.4.7
	github.com/fsouza/go-dockerclient v1.6.6
	github.com/go-errors/errors v1.1.1
	github.com/go-git/go-git/v5 v5.2.0
	github.com/go-ole/go-ole v1.2.4
	github.com/go-sql-driver/mysql v1.5.0
	github.com/goamz/goamz v0.0.0-20180131231218-8b901b531db8
	github.com/gocarina/gocsv v0.0.0-20200925213129-04be9ee2e1a2
	github.com/gogs/chardet v0.0.0-20191104214054-4b6791f73a28
	github.com/golang-collections/go-datastructures v0.0.0-20150211160725-59788d5eb259
	github.com/golang/protobuf v1.4.2
	github.com/golang/snappy v0.0.2
	github.com/google/codesearch v1.2.0
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/sessions v1.2.1
	github.com/hashicorp/go-version v1.2.1
	github.com/hashicorp/golang-lru v0.5.4
	github.com/jaytaylor/html2text v0.0.0-20200412013138-3577fbdbcff7
	github.com/jbussdieker/golibxml v0.0.0-20190103165431-90c340ae5026
	github.com/jinzhu/gorm v0.0.0-20150226080826-33e6b1463252
	github.com/jmoiron/sqlx v1.2.0
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0
	github.com/kennygrant/sanitize v1.2.4
	github.com/kevinburke/ssh_config v0.0.0-20201106050909-4977a11b4351 // indirect
	github.com/kiteco/fsevents v0.0.0-20200103230320-da04af21be99
	github.com/kiteco/go-get-proxied v0.0.0-20191030142721-2003844243fc
	github.com/kiteco/go-porterstemmer v1.0.1
	github.com/kiteco/go-tree-sitter v0.0.0-20201109174848-7a511b990766
	github.com/kiteco/kiteco/kite-go/client/datadeps v0.0.0-00010101000000-000000000000
	github.com/kiteco/tensorflow v0.0.0-20201124180144-caad55e15f15
	github.com/kiteco/websocket v1.8.7-0.20200610174823-316085a5aeb4
	github.com/klauspost/compress v1.11.3 // indirect
	github.com/klauspost/cpuid v1.3.1
	github.com/kr/pretty v0.2.1
	github.com/krolaw/xsd v0.0.0-20190108013600-03ca754cf4c5
	github.com/lib/pq v1.8.0
	github.com/lxn/win v0.0.0-20201021184640-9c48443b1f44
	github.com/lytics/slackhook v0.0.0-20160630154540-a52fd449b27d
	github.com/mattn/go-sqlite3 v1.14.4
	github.com/mattn/go-zglob v0.0.3
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/minio/highwayhash v1.0.1
	github.com/mitchellh/cli v1.1.2
	github.com/mitchellh/go-ps v1.0.0
	github.com/moby/term v0.0.0-20200915141129-7f0af18e79f2 // indirect
	github.com/montanaflynn/stats v0.6.3
	github.com/noahlt/go-mixpanel v0.0.0-20151120050335-9b5b177cf321
	github.com/nwaples/rardecode v1.1.0 // indirect
	github.com/olekukonko/tablewriter v0.0.4 // indirect
	github.com/pierrec/lz4 v2.5.2+incompatible // indirect
	github.com/pkg/errors v0.9.1
	github.com/quickemailverification/quickemailverification-go v1.0.2
	github.com/rollbar/rollbar-go v1.2.0
	github.com/satori/go.uuid v1.2.1-0.20181028125025-b2ce2384e17b
	github.com/sbwhitecap/tqdm v0.0.0-20170314014342-7929e3102f57
	github.com/segmentio/backo-go v0.0.0-20200129164019-23eae7c10bd3 // indirect
	github.com/sergi/go-diff v1.1.0
	github.com/shirou/gopsutil v2.20.2+incompatible
	github.com/shirou/w32 v0.0.0-20160930032740-bb4de0191aa4
	github.com/shurcooL/githubv4 v0.0.0-20200928013246-d292edc3691b
	github.com/shurcooL/graphql v0.0.0-20200928012149-18c5c3165e3a // indirect
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/sourcegraph/annotate v0.0.0-20160123013949-f4cad6c6324d // indirect
	github.com/sourcegraph/syntaxhighlight v0.0.0-20170531221838-bd320f5d308e
	github.com/spf13/afero v1.4.1
	github.com/spf13/cobra v1.1.1
	github.com/ssor/bom v0.0.0-20170718123548-6386211fdfcf // indirect
	github.com/stretchr/testify v1.6.1
	github.com/stripe/stripe-go/v72 v72.37.0
	github.com/syndtr/goleveldb v1.0.0
	github.com/texttheater/golang-levenshtein/levenshtein v0.0.0-20200805054039-cae8b0eaed6c
	github.com/tinylib/msgp v1.1.5
	github.com/ulikunitz/xz v0.5.8 // indirect
	github.com/vaughan0/go-ini v0.0.0-20130923145212-a98ad7ee00ec // indirect
	github.com/wcharczuk/go-chart v2.0.1+incompatible
	github.com/winlabs/gowin32 v0.0.0-20201021184923-54ddf04f16e6
	github.com/xanzy/ssh-agent v0.3.0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonschema v1.2.0
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	github.com/xtgo/uuid v0.0.0-20140804021211-a0b114877d4c
	github.com/yl2chen/cidranger v1.0.2
	github.com/ziutek/mymysql v1.5.4 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20201124201722-c8d3bf9c5392
	golang.org/x/net v0.0.0-20201201195509-5d6afe98e0b7
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43
	golang.org/x/sys v0.0.0-20201201145000-ef89a241ccb3
	golang.org/x/text v0.3.3
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0
	golang.org/x/tools v0.0.0-20201022035929-9cf592e881e9
	gonum.org/v1/plot v0.8.0
	google.golang.org/grpc v1.33.1
	google.golang.org/protobuf v1.25.0
	gopkg.in/gorp.v1 v1.7.2
	gopkg.in/segmentio/analytics-go.v3 v3.1.0
	gopkg.in/yaml.v1 v1.0.0-20140924161607-9f9df34309c0
	gopkg.in/yaml.v2 v2.3.0
	gorm.io/driver/postgres v1.0.8
	gorm.io/gorm v1.21.3
)

replace github.com/fsnotify/fsevents => github.com/kiteco/fsevents v0.0.0-20200103230320-da04af21be99

replace github.com/sirupsen/logrus => github.com/Sirupsen/logrus v1.7.0

replace github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.7.0

replace github.com/kiteco/kiteco/osx => ./osx

replace github.com/kiteco/kiteco/linux => ./linux

replace github.com/kiteco/kiteco/windows => ./windows

replace github.com/kiteco/kiteco/kite-go/client/datadeps => ./kite-go/client/datadeps

replace github.com/rapid7/go-get-proxied => github.com/kiteco/go-get-proxied v0.0.0-20191030142721-2003844243fc
