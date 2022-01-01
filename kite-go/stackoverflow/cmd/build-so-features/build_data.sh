pages=$1
dataDir=$2

mkdir $dataDir

# build so tag synonyms 
cd ~/go/src/github.com/kiteco/kiteco/kite-go/stackoverflow/cmd/so-tags-synonyms
synPath="$dataDir/so-synonyms"
go run *.go --out $synPath

# build tag classes
cd ~/go/src/github.com/kiteco/kiteco/kite-go/stackoverflow/cmd/so-build-tag-classes
tagsPath="$dataDir/tagData"
go run *.go --syn $synPath --pages $pages --out $tagsPath

# build term counters
cd ~/go/src/github.com/kiteco/kiteco/kite-go/stackoverflow/cmd/so-build-counters
countsPath="$dataDir/docCounts"
go run *.go --pages $pages --out $countsPath