from gevent.monkey import patch_all; patch_all()  # uses geventmp to make multiprocessing work
import ast
import base64
import contextlib
import enum
import functools
import gevent
import grp
import http.client
import io
import json
import locale
import mimetypes
import operator
import os
import pwd
import shutil
import signal
import stat
import sys
import traceback
import types
import requests
from typing import AnyStr, Optional, Tuple, List, NamedTuple

import tree_format
import asttokens
from plumbum import local as local_sh


class ColumnTree(NamedTuple):
    data: List[str]
    children: Optional[List['ColumnTree']]

    def format_children(self, *, ralign: Tuple[int, ...] = (), sep: str = ' '):
        widths = [0]*len(self.data)
        for child in self.children:
            for i, s in enumerate(child.data):
                if len(s) > widths[i]:
                    widths[i] = len(s)

        fmts = []
        for i, width in enumerate(widths):
            if i in ralign:
                fmts.append('{:>'+ str(width) + '}')
            else:
                fmts.append('{:'+ str(width) + '}')
        fmt = sep.join(fmts)

        return [(fmt.format(*child.data), child.format_children()) for child in self.children]

    def format(self, *, ralign: Tuple[int, ...] = (), sep: str = ' '):
        return tree_format.format_tree(
            (sep.join(self.data), self.format_children(ralign=ralign, sep=sep)),
            format_node=operator.itemgetter(0),
            get_children=operator.itemgetter(1)
        )


class FileInfo:
    def __init__(self, path: str):
        self.path = path
        self.__stat_info = None

    @property
    def _stat(self) -> os.stat_result:
        if self.__stat_info is None:
            self.__stat_info = os.lstat(self.path)
        return self.__stat_info

    @property
    def _is_dir(self) -> bool:
        return stat.S_ISDIR(self._stat.st_mode)

    @property
    def _is_link(self) -> bool:
        return stat.S_ISLNK(self._stat.st_mode)

    @property
    def name(self) -> str:
        return os.path.basename(self.path)

    @property
    def permissions(self) -> str:
        mode = self._stat.st_mode

        if self._is_dir:
            perms = 'd'
        elif self._is_link:
            perms = 'l'
        else:  # stat.S_ISREG(mode)
            perms = '-'

        for who in "USR", "GRP", "OTH":
            for what in "R", "W", "X":
                bit = '-'
                if mode & getattr(stat, "S_I" + what + who):
                    bit = what.lower()
                perms += bit

        return perms

    @property
    def user(self) -> str:
        try:
            return pwd.getpwuid(self._stat.st_uid).pw_name
        except KeyError:
            return str(self._stat.st_uid)

    @property
    def group(self) -> str:
        try:
            return grp.getgrgid(self._stat.st_gid).gr_name
        except KeyError:
            return str(self._stat.st_gid)

    @property
    def size(self) -> str:
        sz = self._stat.st_size

        if sz < 1024:
            return str(sz) + 'B'
        for unit in ['K','M','G','T','P','E','Z']:
            sz /= 1024.0
            if abs(sz) < 1024.0:
                return "{:3.1f}".format(sz) + unit
        return "{:.1f}Y".format(sz)

    @property
    def list(self):
        if not self._is_dir:
            return
        names = os.listdir(self.path)
        names.sort(key=functools.cmp_to_key(locale.strcoll))
        return [self.__class__(os.path.join(self.path, name)) for name in names]

    def tree(self, *, cols: Tuple[str, ...] = ('permissions', 'size', 'name'), depth: int = 1) -> ColumnTree:
        children = []

        if depth and self._is_dir:
            for child in self.list:
                children.append(child.tree(cols=cols, depth=depth-1))

        data = []
        for col in cols:
            data.append(str(getattr(self, col)))

        return ColumnTree(children=children, data=data)


def remove_trailing_newline(s: str):
    if s and s[-1] == '\n':
        return s[:-1]
    return s


class DisplayCodeState(enum.Enum):
    HIDE = 0
    DISPLAY_NONEMPTY = 1
    DISPLAY = 2



class Runtime:
    __NAME = 'KITE'
    __devnull = open(os.devnull, 'w')
    __sample_files_dir = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'sample-files')


    @staticmethod
    def __format_http_request(req):
        return '{method} {url}\n{headers}\n\n{body}'.format(
            method=req.method,
            url=req.url,
            headers='\n'.join('{}: {}'.format(k, v) for k, v in req.headers.items()),
            body=req.body or '',
        ).strip()

    @staticmethod
    def __format_http_response(res):
        return 'HTTP {status_code} {status_code_text}\n{headers}\n\n{body}'.format(
            status_code=res.status_code,
            status_code_text=http.client.responses.get(res.status_code, ''),
            headers='\n'.join('{}: {}'.format(k, v) for k, v in res.headers.items()),
            body=res.content.decode('utf-8') or '',
        ).strip()

    def __init__(self, filename):
        self.__interactive_mode = False
        self.__auto_io = True
        self.__capture_io = True
        self.__display_code = DisplayCodeState.DISPLAY_NONEMPTY
        self.__blocks = []
        self.__stdout, self.__stderr = io.StringIO(), io.StringIO()
        self.__filename = filename

        self.__async_queue = None
        self.__http_request = None

        # create a fresh __main__ module to simulate running `python <filename>`
        self.__exec_namespace = types.ModuleType("__main__")
        sys.modules["__kite__"] = sys.modules["__main__"]  # put this module under the `__kite__` key
        sys.modules["__main__"] = self.__exec_namespace
        self.__exec_namespace.__dict__[Runtime.__NAME] = self

        # pretend the args start with the filename
        sys.argv = sys.argv[1:]
        sys.stdout = self.__stdout
        sys.stderr = self.__stderr

    def __execute(self, stmt):
        if self.__interactive_mode:
            node = ast.copy_location(ast.Interactive([stmt]), stmt)
            code = compile(node, self.__filename, 'single')
        else:
            node = ast.copy_location(ast.Module([stmt]), stmt)
            code = compile(node, self.__filename, 'exec')

        try:
            exec(code, self.__exec_namespace.__dict__)
        except SystemExit:
            raise
        except:
            etype, value, tb = sys.exc_info()
            traceback_strs = traceback.format_list([frame for frame in traceback.extract_tb(tb) if frame[0] != __file__])
            exception_strs = traceback.format_exception_only(etype, value)
            print("Traceback (most recent call last):\n{}\n{}".format('\n'.join(traceback_strs), '\n'.join(exception_strs)), file=sys.stderr)
            raise  # interrupt execution as if this is the regular Python runtime

    def __execute_all(self, q):
        for stmt in q:
            self.__execute(stmt)

    def _execute(self, stmt: ast.stmt):
        if self.__async_queue is not None:
            self.__async_queue.put(stmt)
        else:
            self.__execute(stmt)

    def _display_code(self, src: str):
        if not src:
            return
        if  self.__display_code == DisplayCodeState.HIDE:
            return
        for code_line in src.split('\n'):
            if code_line.lstrip().startswith(self.__NAME + '.'):
                continue
            if self.__display_code == DisplayCodeState.DISPLAY_NONEMPTY:
                if not code_line.strip():
                    continue
                else:
                    self.__display_code = DisplayCodeState.DISPLAY

            self.__blocks.append({'code_line': code_line.rstrip()})

    def _collect_blocks(self):
        if self.__auto_io:
            self.display_io()
        outputs = self.__blocks
        self.__blocks = []
        return outputs

    def _cleanup(self):
        if self.__http_request:
            display_request, args, kwargs = self.__http_request
            res = requests.request(*args, **kwargs)
            if display_request:
                self.display("HTTP Request", self.__format_http_request(res.request))
            self.display("HTTP Response", self.__format_http_response(res))
        self.display_io()

    def do_async_http(self, *args, display_request: bool = True, **kwargs):
        self.set_capture_io(False)
        self.__async_queue = gevent.queue.Queue()
        gevent.spawn(lambda: self.__execute_all(self.__async_queue))
        self.__http_request = (display_request, args, kwargs)

    def set_capture_io(self, val: bool = True):
        self.display_io()
        self.__capture_io = val
    def set_interactive_mode(self, val: bool = True):
        self.__interactive_mode = val
    def set_auto_io(self, val: bool = True):
        self.__auto_io = val

    def set_display_code(self, val: bool = True):
        if self.__display_code == DisplayCodeState.HIDE:
            if val:
                self.__display_code = DisplayCodeState.DISPLAY_NONEMPTY
        elif not val:
            self.__display_code = DisplayCodeState.HIDE

    def start_prelude(self):
        self.set_display_code(False)
    def stop_prelude(self):
        self.set_display_code(True)

    def display_io(self):
        # collect output
        if self.__capture_io:
            self.display("Output", self.__stdout.getvalue())
            self.display("Error", self.__stderr.getvalue())

        # we don't create new StringIO objects because the existing ones may still be pointed at by sys.stdout/stderr
        # so clear by seeking *and* truncating
        self.__stdout.seek(0)
        self.__stdout.truncate(0)
        self.__stderr.seek(0)
        self.__stderr.truncate(0)

    def display(self, title: str, data: str, type: str = "text"):
        if type == "text":
            data = remove_trailing_newline(data)
        if not data:
            return
        self.__blocks.append({"output": {"type": type, "title": title, "data": data}})

    def list_directory(self,
            path: str ='.',
            *,
            cols: Tuple[str, ...] = ('permissions', 'size', 'name'),
            depth: int = 1,
            ralign: Tuple[str, ...] = ('size',),
            sep: str = ' ',
            title: str = "Directory Listing"):

        ralign = tuple(cols.index(val) for val in ralign if val in cols)
        tree = FileInfo(path).tree(cols=cols, depth=depth)
        if depth == 1:
            data = '\n'.join([x[0] for x in tree.format_children(ralign=ralign, sep=sep)])
        else:
            data = tree.format(ralign=ralign, sep=sep)
        self.display(title, data)

    def display_image(self, path: str, title: Optional[str] = None):
        typ, _ = mimetypes.guess_type(path)
        if type is None:
            raise Exception("could not determine MIME type from image file path")
        with open(path, 'rb') as f:
            data = base64.b64encode(f.read())

        if title is None:
            title = path
        self.display(title, 'data:{};base64,{}'.format(typ, data.decode('utf-8')), type='image')

    def display_file(self, path: str, limit: int = 10000, title: Optional[str] = None):
        with open(path) as f:
            data = f.read()
        if limit > 0 and len(data) > limit:
            data = data[:limit]

        if title is None:
            title = path
        self.display(title, data)

    def write_file(self, path: str, data: AnyStr, encoding: str = 'ascii', display: bool = False):
        if isinstance(data, bytes):
            bdata = data
        elif encoding == 'base64':
            bdata: bytes = base64.b64decode(data)
        else:
            bdata = bytes(data, encoding)

        dir = os.path.dirname(path)
        if dir != '':
            os.makedirs(dir, exists_ok=True)

        with open(path, 'wb') as f:
            f.write(bdata)

        if display:
            self.display_file(path)

    def write_sample_file(self, name: str, dst: str = None, display: bool = False):
        src = os.path.join(self.__sample_files_dir, name)
        if dst is None:
            dst = name

        dir = os.path.dirname(dst)
        if dir != '':
            os.makedirs(dir, exist_ok=True)

        if os.path.isdir(src):
            shutil.copytree(src, dst)
        else:
            shutil.copy(src, dst)

        if display:
            self.display_file(dst)


def main():
    filename = sys.argv[1]

    runtime = Runtime(filename)

    with open(filename) as f:
        src = f.read()
    parsed = asttokens.ASTTokens(src, parse=True, filename=filename)

    cur_pos = 0
    for stmt in parsed.tree.body:
        # we want to emit each (full) line of the source until we've reached the end of the statement
        new_pos = stmt.last_token.endpos
        while new_pos < len(src) and src[new_pos] != '\n':
            # seek to newline
            new_pos += 1

        runtime._display_code(src[cur_pos:new_pos])
        cur_pos = new_pos + 1 # skip the newline from the seek

        try:
            runtime._execute(stmt)
        except:
            # TODO(naman) normally if this exception were uncaught,
            # it would affect the process return code; but for now we ignore that.
            break

        for block in runtime._collect_blocks():
            print(json.dumps(block), file=sys.__stdout__)

    runtime._cleanup()
    for block in runtime._collect_blocks():
        print(json.dumps(block), file=sys.__stdout__)


if __name__ == "__main__":
    try:
        main()
    except:
        # must be a error in the Runtime, not the user's code, so reset IO and re-raise;
        # the Python runtime will handle printing it.
        sys.stdout = sys.__stdout__
        sys.stderr = sys.__stderr__
        raise
