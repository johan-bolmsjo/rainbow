# Rainbow

Rainbow is a log file colorer that act as a stream processor.
See README.md for an overview.

## Rules of Engagement

### Variable, types and function naming

- Don't abbreviate names, except in small scopes such as loop variables.

### Code comment style

- Don't use abbreviations.
- Document the current behavior:
- Use self contained comments: don't refer to external documents or task instructions in comments.
- Don't mention historical details or circumstances.

### Design methodologies

- Use consistent naming of functions, types and variables.
- Apply separation of concern: Consider the responsibilities of code modules for clear interfaces.
- Keep code "DRY": Avoid excessive copy paste.
- Avoid inline magic constants: give constant names to provide context.

### Exempt abbreviations

Local state in small scopes:
- i, j, k (loop variables, indexes)
- err (error)
- k, v (key, value)
- l (line)
- n (counts)
- re (compiled regexp)

Math:
- min (minimum)
- max (maximum)

Data management:
- buf (buffer)
- ctx (context)
- dst (destination)
- msg (message)
- src (source)
- str (string)

Comments:
- e.g. (exempli gratia: for example)
- i.e. (id est: that is)
- API (application programming interface)
