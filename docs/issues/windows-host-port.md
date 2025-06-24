# Windows limitation: host:port names

Using model names that include a host and port (e.g. `localhost:5000/library/model:tag`) fails on Windows. The server stores models on disk using the name as a directory structure. Windows paths cannot contain `:` in directory names which makes these names invalid.

Until this is addressed, avoid using a port in the host portion of model names on Windows.
