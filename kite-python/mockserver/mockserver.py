#!/usr/bin/env python
# -*- coding: utf-8 -*-

import os
import sys
import time
import json
from flask import Flask, request, redirect, url_for

# This server is accessed by code examples to demonstrate the use of http client libraries
# like urllib, requests, etc.

app = Flask(__name__)

short_quote = "Thus was the toll inflicted, the price one paid for being unwittingly born."

quote = \
"""We want to establish the idea that a computer language is not just 
a way of getting a computer to perform operations but rather that it 
is a novel formal medium for expressing ideas about methodology."""

long_quote = \
"""Nor would I have you to mistake in the point of your own liberty.
There is a liberty of corrupt nature, which is affected both by men
and beasts, to do what they list; and this liberty is inconsistent
with authority, impatient of all restraint; by this liberty, Sumus
Omnes Deteriores; tis the grand enemy of truth and peace, and all the
ordinances of God are bent against it. But there is a civil, a moral,
a federal liberty, which is the proper end and object of authority;
it is a liberty for that only which is just and good; for this liberty
you are to stand with the hazard of your very lives.“"""

html_snippet = \
"""<!doctype html>
<html>
	<head>
		<title>Quote</title>
	</head>
	<body>
		<p>If you fail to achieve a correct answer, it is futile to protest that you acted with propriety.</p>
	</body>
</html>"""

long_html_snippet = \
"""<!doctype html>
<html>
	<head>
		<title>Quotes</title>
	</head>
	<body>
		<em>On acting</em>
		<p>The primary thing when you take a sword in your hands is your
		intention to cut the enemy, whatever the means. Whenever you parry,
		hit, spring, strike or touch the enemy’s cutting sword, you must
		cut the enemy in the same movement. It is essential to attain this.
		If you think only of hitting, springing, striking or touching the
		enemy, you will not be able actually to cut him. More than anything,
		you must be thinking of carrying your movement through to cutting 
		him.</p>
		<em>On humans</em>
		<p>Far from being the smartest possible biological species, we are
		probably better thought of as the stupidest possible biological 
		species capable of starting a technological civilization.<p>
	</body>
</html>
"""

jsondata = {
	"page": 40,
	"quote": "A nation can offer huge fortunes and great misery."
}

xmldata = """<?xml version="1.0" encoding="UTF-8"?>
<quote>
	<page>1</page>
	<content>That which can be destroyed by the truth should be.</content>
</quote>"""

article = {
	"title": "Sand",
	"content": "Her mother, buried. A town, lost. A small group of men, somewhere, out there, cheering."
}


@app.route("/shorttext")
def handle_shorttext():
    return short_quote


@app.route("/text")
def handle_text():
    return quote


@app.route("/longtext")
def handle_longtext():
    return long_quote


@app.route("/html")
def handle_html():
    return html_snippet


@app.route("/longhtml")
def handle_longhtml():
    return long_html_snippet


@app.route("/json")
def handle_json():
    return json.dumps(jsondata)


@app.route("/xml")
def handle_xml():
    return xmldata


@app.route("/echo", methods=["GET", "POST"])
def handle_echo():
    return "I received a %s request...\n%s" % (request.method, request.get_data())


@app.route("/status/<status>", methods=["GET", "POST"])
def handle_status(status):
    return "", int(status)


@app.route("/useragent")
def handle_useragent():
	return "Hey there, %s" % request.headers.get('User-Agent', '<no user agent received>')


@app.route("/queryparams")
def handle_queryparams():
	lines = [key + "=" + val for key, val in request.args.items()]
	return "Received the following query parameters:\n" + "\n".join(lines)


@app.route("/submitform", methods=["POST"])
def handle_submitform():
	lines = ["%s = %s\n" % pair for pair in request.form.items()]
	return "Received a form with the following values:\n" + "".join(lines)


@app.route("/submitfiles", methods=["POST"])
def handle_submitfiles():
	lines = ["%s (%d bytes)\n" % (name, len(f.read())) for name, f in request.files.items()]
	return "Received the following files:\n" + "".join(lines)


@app.route("/redirect")
def handle_redirect():
	return redirect("/text")


@app.route("/api/article/<id>", methods=["GET", "POST", "HEAD", "PUT", "DELETE", "PATCH"])
def handle_api(id):
	if request.method == "GET":
		data = article.copy()
		data["id"] = id
		return json.dumps(data)
	elif request.method == "POST":
		return "Added article with ID %d" + str(id)
	elif request.method == "HEAD":
		return ""
	elif request.method == "PUT":
		return "Updated article with ID " + str(id)
	elif request.method == "DELETE":
		return "Deleted article with ID " + str(id)
	elif request.method == "PATCH":
		return "Updated article with ID " + str(id)
	else:
		return "Method not allowed: "+request.method, 405


if __name__ == '__main__':
	port = os.environ.get("PORT", "80")
	app.run(host="0.0.0.0", port=int(port))
