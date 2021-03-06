# UXID

![MIT License][badge_license_url]


**U**ser e**X**perience focused **ID**entifiers (UXIDs) are identifiers which:

* Describe the resource (aid in debugging and investigation)
* Work well with copy and paste (double clicking selects the entire ID)
* Can be shortened for low cardinality resources
* Are secure against enumeration attacks
* Can be generated by application code (not tied to the datastore)
* Are K-sortable (lexicographically sortable by time - works well with datastore indexing)
* Do not require any coordination (human or automated) at startup, or generation
* Are very unlikely to collide (more likely with less randomness)
* Are easily and accurately transmitted to another human using a telephone

Many of the concepts of Stripe IDs have been used in this library.

## Usage

#### Generating UXIDs via the CLI

```bash
go run cli/main.go --prefix=cus --size=small # cus_01ER59H9BVY1R0
go run cli/main.go --prefix=txn --size=xl    # txn_01ER59HZ556GREFZ57T1RNYV21
```

## Contributing

Bug reports and pull requests are welcome on GitHub at https://github.com/riddler/uxid-go.


## License

The project is available as open source under the terms of the [MIT License](https://opensource.org/licenses/MIT).

<!-- LINKS -->
[mit_license_url]: http://opensource.org/licenses/MIT

<!-- BADGES -->
[badge_license_url]: https://img.shields.io/badge/license-MIT-brightgreen.svg?cacheSeconds=3600?style=flat-square
