import logging
import pkg_resources
import re

logger = logging.getLogger(__name__)

# https://www.python.org/dev/peps/pep-0503/#normalized-names
_normalize_separator = re.compile(r"[-_.]+")
def normalize_name(name):
    return _normalize_separator.sub("-", name).lower()


class DistMeta(object):
    """
    DistMeta represents a Python distribution's metadata. All names are normalized.
    """

    def __init__(self, dist):
        """
        DO NOT CALL THIS METHOD, USE from_requirement_string.

        Note: we do not support distutils packages.

        @param dist: The underlying pkg_resources.Distribution object to extract the information from.
        """
        self._dist = dist

    @property
    def name(self):
        return normalize_name(self._dist.project_name)

    @property
    def version(self):
        return self._dist.version

    @property
    def requires(self):
        for req in self._dist.requires():
            meta = self.from_name(req.name)
            if meta is None:
                continue
            yield meta

    def dependencies(self):
        """
        Calculates and returns the transitive closure of the distribution's dependencies, including the distribution itself, ignoring versions.
        """
        deps = {}

        # bfs with cycle-detection
        queue = [self]
        while len(queue) > 0:
            dep = queue.pop(0)
            if dep.name in deps:
                continue
            deps[dep.name] = dep
            yield dep
            queue.extend(dep.requires)

    @property
    def top_level_names(self):
        names = set()

        if self._dist.has_metadata("PKG-INFO"):
            for line in self._dist.get_metadata_lines("PKG-INFO"):
                if line.startswith("Name:"):
                    names.add(line.lstrip("Name:").strip())
                if line.startswith("Provides:"):
                    names.update(name.strip() for name in line.lstrip("Provides:").split())

        if self._dist.has_metadata("top_level.txt"):
            for line in self._dist.get_metadata_lines("top_level.txt"):
                names.update(part.strip() for part in line.split())

        return names

    @classmethod
    def from_name(cls, name):
        """
        Create a Package object from a requirement string.

        Note: we do not support distutils packages.

        @param req_str: the requirement string.
        """
        name = normalize_name(name)
        for candidate, dist in pkg_resources.working_set.by_key.items():
            if normalize_name(candidate) != name:
                continue
            return cls(dist)
        return None
