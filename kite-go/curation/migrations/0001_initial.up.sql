CREATE TABLE "Comment" (
  "ID" bigint(20) NOT NULL,
  "SnippetID" bigint(20),
  "Text" varchar(255),
  "Created" bigint(20),
  "CreatedBy" varchar(255),
  "Modified" bigint(20),
  "ModifiedBy" varchar(255),
  "Dismissed" bigint(20),
  "DismissedBy" varchar(255),
  PRIMARY KEY ("ID")
);

CREATE TABLE "CuratedSnippet" (
  "ID" bigint(20) unsigned NOT NULL,
  "Language" varchar(255),
  "Package" varchar(255),
  "Title" text,
  "Prelude" text,
  "Code" text,
  "Postlude" text,
  "Output" text,
  "Deleted" bigint(20) NOT NULL,
  "DeletedBy" varchar(255) NOT NULL DEFAULT '',
  "Created" bigint(20) NOT NULL,
  "CreatedBy" varchar(255) NOT NULL DEFAULT '',
  "Modified" bigint(20) NOT NULL,
  "ModifiedBy" varchar(255) NOT NULL DEFAULT '',
  "Imported" bigint(20) NOT NULL,
  "ImportedBy" varchar(255) NOT NULL DEFAULT '',
  "DisplayOrder" bigint(20) NOT NULL,
  PRIMARY KEY ("ID")
);

CREATE TABLE "CuratedSnippetTag" (
  "SnippetID" bigint(20),
  "Name" varchar(255) 
);

CREATE TABLE "EmphasizedSegment" (
  "SnippetID" bigint(20) unsigned,
  "Begin" int(11),
  "End" int(11) 
);

CREATE TABLE "ExampleOf" (
  "SnippetID" bigint(20) unsigned,
  "FunctionName" varchar(255) 
);

CREATE TABLE "Related" (
  "SnippetID" bigint(20) unsigned,
  "RelatedID" bigint(20) unsigned 
);

CREATE TABLE "User" (
  "ID" bigint(20) NOT NULL,
  "Name" varchar(255),
  "PasswordHash" varchar(255),
  PRIMARY KEY ("ID")
);
