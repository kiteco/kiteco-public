rm db.sqlite3
rm -rf blog/migrations
python manage.py makemigrations blog
python manage.py migrate
python manage.py loaddata articles_complex