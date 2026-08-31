# Feature 0002 — The base game loop

Architecture in `docs/adr/0013-harvest-economy.md` and `docs/adr/0012-named-strains.md`. This is the loop as a player meets it.

## The loop

**Buy a seed → sow it → wait → gather it → get paid → buy another.**

A new farm starts with three Stem seeds and no cash, so the first move is to sow.

## Working the farm

One gesture does everything: **click a plot**.

- Empty plot, seed queued → sows it.
- Growing plot → selects it, so the panel shows what it is and how far along.
- Ready plot → gathers it.

A queued seed stays queued while stock lasts, so sowing a row is one click per plot. A message under the panel reports what happened.

## The Stem

The starter crop: a plain upright stalk, no flower, no fruit. It cannot grow into anything else — the species clamps its archetype genes — but its height, thickness, colour, leaf shape, growth rate, yield and **density** all vary.

It is deliberately forgiving. A Stem is always hardy and almost always harvests successfully, so a new player is never punished by dice. It is always an annual, so the plot clears and the loop repeats.

**A starter Stem matures in 20 seconds.** Growth Rate varies that a lot: the fastest Stem the gene allows ripens in 10 seconds, the slowest takes 75. Speed is worth breeding for in its own right, not just density.

## Getting paid

A harvest pays `units × unit value`.

- **Yield Amount** sets how many units.
- **Yield Quality** raises unit value.
- **Density** raises it four times harder. Density is the stat the whole game is climbing toward, so it is the one worth breeding for.

A harvest can fail. **Harvest Chance** decides it, and a failure loses the crop for nothing. The Stem's floor is high enough that this is rare.

## Seeds and drift

A harvest returns seeds carrying the parent's genome. **A seed is very nearly a clone**: it gets a single chance of a copy error, about one in ten thousand, and otherwise comes back identical. Sowing and gathering a line will not, on its own, change it.

Returns average a little under one seed per harvest, so a farm slowly runs down and the shop stays worth visiting.

To make a line move you buy into it. The **Seed Irradiator** makes mutations twelve times as likely, and the **Reactor Bed** twelve times likelier again; a plant's own **Mutability** gene multiplies that further. With both upgrades and a mutable line, drift becomes a real generation-by-generation process. Without them, your stems breed true.

## The barn

Seeds are listed one row per species — `Stem  7 lines  x24` — so ordinary drift never fills the panel. **Named strains get their own row**, in their rarity colour, because those are the ones worth picking out.

Clicking a species row sows your **bulk line** — the one you hold most of — so sowing a field is one click per plot and never stops to ask which near-identical seed you meant.

A row holding more than one line carries two buttons:

- **The list icon** opens the **seed index**: every line you hold, drawn as the plant it grows into, with its Density, Growth and Yield, and buttons to sow it or discard it. Discarding takes the whole line.
- **The tick** toggles auto-pick. Lit, the row sows your bulk line. Unlit, clicking the row opens the index instead, for when you are working through several lines deliberately. The setting is remembered per crop.

## Named strains

Some plants are worth recognising. A strain is named one of two ways.

**Bred strains** are defined by conditions on visible traits, and each states its goal, so they are things to aim at rather than things to stumble on:

| Strain | Goal | Rarity |
|---|---|---|
| Sunspire | Near-maximum Stem Height and Growth Rate | Uncommon |
| Palewood | Very low Stem Saturation with high Luminance | Uncommon |
| Ironstem | Very high Density on a very thick stem | Rare |
| Gnarlroot | A hard-bent stem kept short | Rare |

**Bought strains** are one exact genome and cannot be bred at all. **Voidshoot** — maximum density, a stalk so dark it drinks light — is sold under licence.

Meeting a strain logs it. The count sits in the header, and the log survives prestige.

## The shop

- **Stem Seed**, $5, always on sale.
- **Field Extension**, $100 — clears another column of ground, taking the farm from 3x3 to 4x3. The usual first goal, reachable within the opening couple of minutes. Anything already growing keeps growing; the new column arrives as bare soil.
- **Voidshoot Licence**, $2,500 — a one-off upgrade that puts the Voidshoot Seed on sale at $900.

A rare seed is gated by an upgrade rather than by its own economy, so unlocking one is an ordinary purchase.

## What is not here yet

Cropsticks and adjacency breeding, expanding the farm, automation, offline progress, prestige, and any crop other than the Stem.
