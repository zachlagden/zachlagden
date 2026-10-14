# Skills graphic

`build.mjs` draws the "What I work with" section of the profile README as two SVG files, one for light themes and one for dark themes. It also rewrites the `<picture>` block between the `skills:start` and `skills:end` markers in `../README.md`, including the alt text that lists every item.

## Update the list

1. Edit `skills.json`. Each item needs a `name` and an `icon` key from `icons.json`. An optional `detail` appears only in the alt text.
2. Run `node skills/build.mjs` from the repository root.
3. Commit the changed SVG files and README.

## Add a logo

Add an entry to `icons.json` with a `viewBox`, a `hex` brand colour (or `null` to use the text colour), the shape markup in `svg`, and its `source` and `licence`. Set `"original": true` to keep a multi-colour logo's own colours.

A logo whose brand colour has less than 3.5:1 contrast against the pill background uses the text colour instead, so light logos stay readable on the light theme and dark logos on the dark theme.

## Logo sources

| Source | Licence | Logos |
| --- | --- | --- |
| [Simple Icons](https://simpleicons.org) 16.35.0 | CC0-1.0 | Most entries |
| [LobeHub icons](https://github.com/lobehub/lobe-icons) | MIT | OpenAI |
| [borgmatic](https://github.com/borgmatic-collective/borgmatic) | Not stated | Borgmatic |
| [mailcow](https://mailcow.email/images/cow_mailcow.svg) | Not stated | Mailcow |

Every logo belongs to its owner and appears only to identify the technology.
