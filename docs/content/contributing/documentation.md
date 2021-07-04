# Documentation

Features Are Better When You Know How to Use Them
{: .subtitle }

You've found something unclear in the documentation and want to give a try at explaining it better?
Let's see how.

## Building Documentation

### General

This [documentation](https://doc.traefik.io/traefik/) is built with [mkdocs](https://mkdocs.org/).

### Method 1: `Docker` and `make`

You can build the documentation and test it locally (with live reloading), using the `docs-serve` target:

```bash
$ make docs-serve
docker build -t traefik-docs -f docs.Dockerfile .
# […]
docker run  --rm -v /home/user/go/github/traefik/traefik:/mkdocs -p 8000:8000 traefik-docs mkdocs serve
# […]
[I 170828 20:47:48 server:283] Serving on http://0.0.0.0:8000
[I 170828 20:47:48 handlers:60] Start watching changes
[I 170828 20:47:48 handlers:62] Start detecting changes
```

!!! tip "Default URL"

    Your local documentation server will run by default on [http://127.0.0.1:8000](http://127.0.0.1:8000).

If you only want to build the documentation without serving it locally, you can use the following command:

```bash
$ make docs-build
...
```

### Method 2: `mkdocs`

First, make sure you have `python` and `pip` installed.

```bash
$ python --version
Python 2.7.2
$ pip --version
pip 1.5.2
```

Then, install mkdocs with `pip`.

```bash
pip install --user -r requirements.txt
```

To build the documentation locally and serve it locally, run `mkdocs serve` from the root directory.
This will start a local server.

```bash
$ mkdocs serve
INFO    -  Building documentation...
INFO    -  Cleaning site directory
[I 160505 22:31:24 server:281] Serving on http://127.0.0.1:8000
[I 160505 22:31:24 handlers:59] Start watching changes
[I 160505 22:31:24 handlers:61] Start detecting changes
```

### Check the Documentation

To check that the documentation meets standard expectations (no dead links, html markup validity, ...), use the `docs-verify` target.

```bash
$ make docs-verify
docker build -t traefik-docs-verify ./script/docs-verify-docker-image ## Build Validator image
...
docker run --rm -v /home/travis/build/traefik/traefik:/app traefik-docs-verify ## Check for dead links and w3c compliance
=== Checking HTML content...
Running ["HtmlCheck", "ImageCheck", "ScriptCheck", "LinkCheck"] on /app/site/basics/index.html on *.html...
```

!!! note "Clean & Verify"

    If you've made changes to the documentation, it's safter to clean it before verifying it.

    ```bash
    $ make docs
    ...
    ```

    Will perform all necessary steps for you.

!!! note "Disabling Documentation Verification"

    Verification can be disabled by setting the environment variable `DOCS_VERIFY_SKIP` to `true`:

    ```shell
    DOCS_VERIFY_SKIP=true make docs-verify
    ...
    DOCS_LINT_SKIP is true: no linting done.
    ```
