name: Bug Report (Traefik)
description: Create a report to help us improve.
body:
  - type: checkboxes
    id: terms
    attributes:
      label: Welcome!
      description: |
        The issue tracker is for reporting bugs and feature requests only. For end-user related support questions, please refer to one of the following:
        - the Traefik community forum: https://community.containo.us/

        The configurations between 1.X and 2.X are NOT compatible. Please have a look [here](https://doc.traefik.io/traefik/getting-started/configuration-overview/).

        DO NOT FILE ISSUES FOR GENERAL SUPPORT QUESTIONS.
      options:
        - label: Yes, I've searched similar issues on [GitHub](https://github.com/traefik/traefik/issues) and didn't find any.
          required: true
        - label: Yes, I've searched similar issues on the [Traefik community forum](https://community.containo.us) and didn't find any.
          required: true

  - type: textarea
    attributes:
      label: What did you do?
      description: |
        How to write a good bug report?

        - Respect the issue template as much as possible.
        - The title should be short and descriptive.
        - Explain the conditions which led you to report this issue: the context.
        - The context should lead to something, an idea or a problem that you’re facing.
        - Remain clear and concise.
        - Format your messages to help the reader focus on what matters and understand the structure of your message, use [Markdown syntax](https://help.github.com/articles/github-flavored-markdown)
      placeholder: What did you do?
    validations:
      required: true

  - type: textarea
    attributes:
      label: What did you see instead?
      placeholder: What did you see instead?
    validations:
      required: true

  - type: textarea
    attributes:
      label: What version of Traefik are you using?
      description: |
        `latest` is not considered as a valid version.

        Output of `traefik version`.

        For the Traefik Docker image (`docker run [IMAGE] version`), example:
        ```console
        $ docker run traefik version
        ```
      placeholder: Paste your output here.
    validations:
      required: true

  - type: textarea
    attributes:
      label: What is your environment & configuration?
      description: arguments, toml, provider, platform, ...
      placeholder: Add information here.
      value: |
        ```yaml
        # (paste your configuration here)
        ```

        Add more configuration information here.
    validations:
      required: true

  - type: textarea
    attributes:
      label: If applicable, please paste the log output in DEBUG level
      description: "`--log.level=DEBUG` switch."
      placeholder: Paste your output here.
    validations:
      required: false
