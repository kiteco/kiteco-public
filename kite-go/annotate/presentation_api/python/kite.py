import os
import stat
import ast
import sys
import imp
import json
import base64
import mimetypes


class kite(object):
	_stream = None

	@classmethod
	def show(cls, **kwargs):
		sys.stdout.write("[[KITE[[SHOW %s]]KITE]]\n" % json.dumps(kwargs))

	@classmethod
	def show_plaintext_str(cls, value, expression=""):
		kite.show(type="plaintext", expression=expression, value=str(value))

	@classmethod
	def show_plaintext(cls, value, expression=""):
		kite.show_plaintext_str(repr(value), expression)

	@classmethod
	def format_permissions(cls, mode):
		masks = (
			(stat.S_IRUSR, 'r'), (stat.S_IWUSR, 'w'), (stat.S_IXUSR, 'x'),
			(stat.S_IRGRP, 'r'), (stat.S_IWGRP, 'w'), (stat.S_IXGRP, 'x'),
			(stat.S_IROTH, 'r'), (stat.S_IWOTH, 'w'), (stat.S_IXOTH, 'x')
			)

		perms = ["-"]*10
		if stat.S_ISDIR(mode):
			perms[0] = 'd'
		for i, (bit, ch) in enumerate(masks):
			if bool(mode & bit):
				perms[i+1] = ch

		return ''.join(perms)

	@classmethod
	def owner_name(cls, uid):
		import pwd
		try:
			u = pwd.getpwuid(uid)[0]
		except KeyError, e:
			u = "-"
		return u or "-"

	@classmethod
	def group_name(cls, gid):
		import grp
		try:
			g = grp.getgrgid(gid)[0]
		except KeyError, e:
			g = "-"
		return g or "-"

	@classmethod
	def dir_entry_data(cls, path, cols=[]):
		info = os.stat(path)
		return {
			"size": -1 if stat.S_ISDIR(info.st_mode) else info.st_size,
			"permissions": kite.format_permissions(info.st_mode),
			"modified": int(info.st_mtime),
			"created": int(info.st_ctime),
			"accessed": int(info.st_atime),
			"ownerid": int(info.st_uid),
			"owner": kite.owner_name(info.st_uid),
			"groupid": int(info.st_gid),
			"group": kite.group_name(info.st_gid)
		}

	@classmethod
	def show_dir_table(cls, path=".", caption=None, cols=None):
		contents = os.listdir(path)

		# default columns (too long to specify as default args in sig)
		if not cols:
			cols = ["size", "permissions", "owner", "group"]

		directory = []
		for entry in contents:
			data = {"name": entry}
			data.update(kite.dir_entry_data(os.path.join(path, entry), cols))
			directory.append(data)

		kite.show(type="dirtable", path=path, caption=caption, cols=cols, entries=directory)

	@classmethod
	def show_dir_tree(cls, path=".", caption=None, skip=None):
		kite.show(type="dirtree", path=path, caption=caption, entries=kite.path_flat(path, skip))

	@classmethod
	def path_flat(cls, path, skip):
		import fnmatch
		do_not_recurse = []
		if skip is not None:
			do_not_recurse = [(p if p.startswith("/") else path+"/"+p) for p in skip]

		entries = dict()
		for root, dirs, files in os.walk(path):
		    for f in files:
		        name = os.path.join(root, f)
		        if not any(fnmatch.fnmatch(name, pat) for pat in do_not_recurse):
		            entries[name] = mimetypes.guess_type(name)[0]

		    to_remove = []
		    for d in dirs:
		        name = os.path.join(root, d)
		        if not any(fnmatch.fnmatch(name, pat) for pat in do_not_recurse):
		            entries[name] = "application/x-directory"
		        else:
		            to_remove.append(d)
		    for d in to_remove:
		        dirs.remove(d)
		return entries

	@classmethod
	def show_image_path(cls, path, caption, encoding=None, data=None):
		# imports are inside functions to avoid polluting the broader code example
		# image annotations work by writing an image to a file, then showing an
		# annotation that says "show this image here"
		if data is None:
			with open(path) as f:
				data = base64.b64encode(f.read())
		else:
			data = base64.b64encode(data)
		kite.show(type="image", path=path, caption=caption, encoding=encoding, data=data)

	@classmethod
	def show_file(cls, path, caption, data=None):
		# imports are inside functions to avoid polluting the broader code example
		if data is None:
			with open(path) as f:
				data = f.read()
		if len(data) > 10000:
			data = data[:10000]
		kite.show(type="file", path=path, caption=caption, data=base64.b64encode(data))

	@classmethod
	def show_region_delimiter(cls, region):
		kite.show(type="region", region=region)
