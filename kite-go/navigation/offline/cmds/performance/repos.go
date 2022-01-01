package main

import (
	"os"
	"path/filepath"

	"github.com/kiteco/kiteco/kite-go/navigation/localpath"
)

var (
	kiteco = localpath.Absolute(os.Getenv("GOPATH")).Join("src", "github.com", "kiteco", "kiteco")
	valDir = localpath.Absolute(os.Getenv("HOME")).Join("nav-validation")

	repos = []repo{
		repo{
			name: "kiteco/kiteco",
			root: kiteco,
			currentPath: filepath.Join(
				os.Getenv("GOPATH"), "src", "github.com", "kiteco", "kiteco",
				"kite-go", "lang", "python", "pythoncomplete", "api", "api.go",
			),
		},
		repo{
			name: "angular/angular",
			root: valDir.Join("angular", "angular", "root"),
			currentPath: filepath.Join(
				os.Getenv("HOME"), "nav-validation", "angular", "angular", "root",
				"packages", "upgrade", "src", "common", "src", "angular1.ts",
			),
		},
		repo{
			name: "apache/airflow",
			root: valDir.Join("apache", "airflow", "root"),
			currentPath: filepath.Join(
				os.Getenv("HOME"), "nav-validation", "apache", "airflow", "root",
				"airflow", "www", "views.py",
			),
		},
		repo{
			name: "apache/hive",
			root: valDir.Join("apache", "hive", "root"),
			currentPath: filepath.Join(
				os.Getenv("HOME"), "nav-validation", "apache", "hive", "root",
				"ql", "src", "test", "org", "apache", "hadoop", "hive", "ql", "io", "orc", "TestVectorizedOrcAcidRowBatchReader.java",
			),
		},
		repo{
			name: "apache/spark",
			root: valDir.Join("apache", "spark", "root"),
			currentPath: filepath.Join(
				os.Getenv("HOME"), "nav-validation", "apache", "spark", "root",
				"sql", "catalyst", "src", "main", "java", "org", "apache", "spark", "sql", "connector", "catalog", "SupportsDelete.java",
			),
		},
		repo{
			name: "django/django",
			root: valDir.Join("django", "django", "root"),
			currentPath: filepath.Join(
				os.Getenv("HOME"), "nav-validation", "django", "django", "root",
				"django", "contrib", "admin", "checks.py",
			),
		},
		repo{
			name: "facebook/react",
			root: valDir.Join("facebook", "react", "root"),
			currentPath: filepath.Join(
				os.Getenv("HOME"), "nav-validation", "facebook", "react", "root",
				"packages", "shared", "ReactComponentStackFrame.js",
			),
		},
		repo{
			name: "prestodb/presto",
			root: valDir.Join("prestodb", "presto", "root"),
			currentPath: filepath.Join(
				os.Getenv("HOME"), "nav-validation", "prestodb", "presto", "root",
				"presto-spark-base", "src", "main", "java", "com", "facebook", "presto", "spark", "PrestoSparkInjectorFactory.java",
			),
		},
		repo{
			name: "rails/rails",
			root: valDir.Join("rails", "rails", "root"),
			currentPath: filepath.Join(
				os.Getenv("HOME"), "nav-validation", "rails", "rails", "root",
				"railties", "lib", "rails", "generators", "actions.rb",
			),
		},
		repo{
			name: "spring-projects/spring-framework",
			root: valDir.Join("spring-projects", "spring-framework", "root"),
			currentPath: filepath.Join(
				os.Getenv("HOME"), "nav-validation", "spring-projects", "spring-framework", "root",
				"spring-messaging", "src", "main", "java", "org", "springframework", "messaging", "simp", "stomp", "StompSession.java",
			),
		},
		repo{
			name: "tensorflow/tensorflow",
			root: valDir.Join("tensorflow", "tensorflow", "root"),
			currentPath: filepath.Join(
				os.Getenv("HOME"), "nav-validation", "tensorflow", "tensorflow", "root",
				"tensorflow", "lite", "core", "api", "flatbuffer_conversions.cc",
			),
		},
	}
)
