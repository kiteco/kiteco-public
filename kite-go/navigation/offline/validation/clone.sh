while read p; do
	git clone --single-branch https://github.com/"$p".git "$1"/"$p"/root
done <$2
