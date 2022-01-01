while read p; do
	git show --pretty="" --name-only $p | sed 's/^/'$p',/g'
done <$1
