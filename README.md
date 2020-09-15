# tbdd: *t*ar*b*all *d*e-*d*uplicator

`tbdd` is a simple tarball de-duplicator. It unpacks a `.tar.gz` file and stores the data as the SHA256 of the contents so that duplicate data is only stored once. It replaces the data with a pointer to the stored data. When the tarball is requested later, it is reconstructed from the metadata and stored data.

Note that the tarball returned by `tbdd` might not be identical to the one stored in it due to compression levels. The contents should be identical though.

# API

## Save a tarball
```
curl --upload-file sample.tar.gz http://tbdd.local/sample.tar.gz
```

## Fetch a tarball
```
curl -s http://tbdd.local/sample.tar.gz | tar xvfz -
```

