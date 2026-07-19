# IEEE OUI Registry

`oui.tsv.gz` is generated from the IEEE Registration Authority registries:

https://standards-oui.ieee.org/oui/oui.csv

https://standards-oui.ieee.org/oui28/mam.csv

https://standards-oui.ieee.org/oui36/oui36.csv

The application embeds this file and performs vendor lookups locally. It does
not make network requests at runtime.

Refresh the bundled registry from the repository root:

```sh
go run ./scripts/update_oui.go
```

The updater keeps the MA-L, MA-M, and MA-S assignment prefixes and organization
names, sorts the records, and writes a compressed tab-separated index. Runtime
lookup uses the longest matching assignment.
