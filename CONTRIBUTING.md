# Contributing

lazyenv is maintained by **Steven Raimondi** ([@lazynop](https://github.com/lazynop)).

## Reporting issues

Found a bug or have a feature request? Open an issue on [GitHub](https://github.com/lazynop/lazyenv/issues).

## Development

### Prerequisites

- Go 1.26+
- [just](https://github.com/casey/just) (task runner)

### Build and test

```bash
just build        # build to bin/lazyenv
just test         # run tests only
just check        # fmt + vet + tests (same as CI)
```

### Before submitting

Run `just fix` to apply Go modernization fixes, then make sure `just check` passes with no errors:

```bash
just fix
just check
```

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
