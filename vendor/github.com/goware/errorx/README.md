[![Build Status](https://travis-ci.org/c2h5oh/errorx.svg?branch=master)](https://travis-ci.org/c2h5oh/errorx)
[![GoDoc](https://godoc.org/github.com/c2h5oh/errorx?status.svg)](https://godoc.org/github.com/c2h5oh/errorx)

# errorx
Feature rich Golang error interface implementation inspired by Postgres error message style guide http://www.postgresql.org/docs/devel/static/error-style-guide.html

# features
* **Filename and line on which error occures** in Debug verbosity level. Not 100% accurate - shows file/line where errorx is rendered to string/JSON, but still quite helpful
* Error codes
* 3 levels of error reporting: Info, Verbose, Debug, each providing more information 
* Everything Golang `error` has
* Everything Golang `errors` package provides
* Formatted errors with parameters
* JSON errors you can just write to your webhandler

# docs
http://godoc.org/github.com/c2h5oh/errorx
