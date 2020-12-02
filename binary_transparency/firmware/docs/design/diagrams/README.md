# Diagrams

This directory contains system design diagrams.

## UML

The UML diagrams are created with the open source [Mermaid](https://mermaid-js.github.io/mermaid/#/)
tool from the `*.mmd` files.

You can use the mermaid-cli tool to render these locally.
To install `mermaid-cli`:

```bash
$ pushd ${HOME} && npm install @mermaid-js/mermaid-cli
# You probably want to make the following change in your .profile too:
$ export PATH=${PATH}:~/node_modules/.bin
```

To render the Mermaid files:

```bash
$ for i in *mmd; do echo $i; mmdc -i $i -o ${i/mmd/svg}; done
```

