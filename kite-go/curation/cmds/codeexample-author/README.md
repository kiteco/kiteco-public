Setup
=====

You should do all this inside the kite-dev vm.  Install mysql, create the curation database, then populate it with a database dump from production:

	$ apt-get install mysql-server
	$ mysql -u root
	mysql> CREATE DATABASE 'curation';
	$ mysqldump -h main.XXXXXXX.us-west-1.rds.amazonaws.com -u labelinguser -p <PASSWORD> labeling | mysql -uroot curation

This way, if anything goes wrong, you can always nuke the database and re-clone the production database.

(You'll have to get <PASSWORD> from the Quip doc: <https://quip.com/XXXXXXX>)


User setup
==========

To set up a new user, you can use curation/cmds/usertool:

	$ go build github.com/kiteco/kiteco/kite-go/curation/cmds/usertool
	$ ./usertool -name="Test" -email="XXXXXXX" -password="XXXXXXX"

Running
=======

To run (serving web assets from memory):

	$ go generate github.com/kiteco/kiteco/kite-go/curation/cmds/codeexample-author
	$ go build github.com/kiteco/kiteco/kite-go/curation/cmds/codeexample-author
	$ ./codeexample-author --port :3000 --db root@/curation

If you're doing front-end development (HTML/CSS/JS), you may find it useful to use dev mode, which serves assets from the filesystem rather than from memory:

	$ go build github.com/kiteco/kiteco/kite-go/curation/cmds/codeexample-author
	$ ./codeexample-author --port :3000 --db root@/curation --dev=/kiteco/kite-go/curation/cmds/codeexample-author

Like the Kite Sidebar, the curation frontend uses LESS and JSX.  To develop on it, first download and run the node.js installer from <http://nodejs.org/download>.

Then install its dependencies:

	$ cd kiteco/kite-go/curation/cmds/codeexample-author
	$ npm install

In development, you should run the LESS and JSX watchers to automatically compile these files:

	$ npm run watch-js
	$ npm run watch-css

You shouldn't need to do this yourself (`go generate` should take care of it), but you can always rebuild the minified Javascript bundle manually:

	$ npm run build
