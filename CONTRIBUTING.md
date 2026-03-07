# Contributing

lazyenv is maintained by **Steven Raimondi** ([@traveltoaiur](https://gitlab.com/traveltoaiur)).

## Reporting issues

Found a bug or have a feature request? Open an issue on [GitLab](https://gitlab.com/traveltoaiur/lazyenv/-/issues).

## Development

### Prerequisites

- Go 1.26+
- [just](https://github.com/casey/just) (task runner)

### Build and test

```bash
just build        # build to bin/lazyenv
just test         # run tests only
just fix          # apply Go modernization fixes
just check        # fmt + vet + tests (same as CI)
```

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
