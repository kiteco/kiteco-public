queries=$1
out=$2

temp_out=~/temp

old_proxy=$HTTP_PROXY

# set http prox for crawlera
export HTTP_PROXY=$CRAWLERA

# get google suggest results
cd ~/go/src/github.com/kiteco/kiteco/kite-go/stackoverflow/cmd/google-so-results

go run *.go --input $queries --outputDir $temp_out

export HTTP_PROXY=$old_proxy

# fetch so pages
cd ~/go/src/github.com/kiteco/kiteco/kite-go/stackoverflow/cmd/fetch-so-pages

go run *.go --in $temp_out --out $out

rm -rf $temp_out
