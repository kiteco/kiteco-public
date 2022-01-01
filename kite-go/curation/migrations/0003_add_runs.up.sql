CREATE TABLE Run (
	ID INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
	SnippetID INTEGER NOT NULL,
	Timestamp INTEGER NOT NULL,
	Stdin BLOB,
	Stdout BLOB,
	Stderr BLOB,
	Succeeded INTEGER NOT NULL,
	SandboxError VARCHAR(255) NOT NULL DEFAULT ''
);

CREATE TABLE HTTPOutput (
	ID INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
	RunID INTEGER NOT NULL,
	RequestMethod VARCHAR(255) NOT NULL DEFAULT '',
	RequestURL VARCHAR(255) NOT NULL DEFAULT '',
	RequestHeaders TEXT,
	RequestBody BLOB,
	ResponseStatus VARCHAR(255) NOT NULL DEFAULT '',
	ResponseStatusCode INTEGER NOT NULL,
	ResponseHeaders TEXT,
	ResponseBody BLOB
);

CREATE TABLE OutputFile (
	ID INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
	RunID INTEGER NOT NULL,
	Path VARCHAR(255) NOT NULL DEFAULT '',
	ContentType VARCHAR(255) NOT NULL DEFAULT '',
	Contents BLOB
);

CREATE TABLE CodeProblem (
	ID INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
	RunID INTEGER NOT NULL,
	Level VARCHAR(255) NOT NULL DEFAULT '',
	Segment VARCHAR(255) NOT NULL DEFAULT '',
	Message TEXT,
	Line INTEGER NOT NULL
);

# Data migration
INSERT INTO Run (SnippetID, Timestamp, Stdout, Succeeded)
	SELECT CuratedSnippet.SnippetID, CuratedSnippet.Modified, CuratedSnippet.Output, 1
	FROM CuratedSnippet
	WHERE CuratedSnippet.SnapshotID IN (
		SELECT MAX(CuratedSnippet.SnapshotID) FROM CuratedSnippet GROUP BY CuratedSnippet.SnippetID);

# Delete old columns
ALTER TABLE CuratedSnippet DROP COLUMN Output
