# Commercial License

> <strong>English</strong> · <a href="./doc/COMMERCIAL.zh.md">繁體中文</a>

Agenvoy is dual-licensed. This page describes the commercial option; for the open source option see [LICENSE](LICENSE).

**For any commercial licensing need, contact the developer directly at `dev@pardn.io`** — there is no reseller, no sales form, and no public price list; terms are agreed per engagement.

## Which one applies to you

| Your situation | License |
|---|---|
| Personal use, research, evaluation | AGPL-3.0 — no action needed |
| Internal tool, not offered to users over a network | AGPL-3.0 — no action needed |
| You release your derivative work under AGPL-3.0, source included | AGPL-3.0 — no action needed |
| You offer Agenvoy, or a work derived from it, to users over a network without publishing your source | **Commercial** |
| You embed Agenvoy in a proprietary product you distribute | **Commercial** |
| Your organisation's policy prohibits AGPL in the delivered product | **Commercial** |

The deciding question is AGPL-3.0 §13: if users interact with a modified version over a network, that version's complete source must be offered to them. A commercial license removes that obligation.

## What the commercial license grants

- Use, modify and distribute Agenvoy without the AGPL-3.0 source-disclosure obligation
- Embed it in closed-source products and hosted services
- Keep your own modifications private
- Remove the **License** and **Source** entries from the web interface, which the AGPL-3.0 section 7(b) additional terms otherwise require you to keep

It does not transfer copyright, and it does not restrict anyone else's use of the AGPL-3.0 version.

## What it does not include

The commercial license grants rights to the software only. It buys no time from the developer.

**Any work on the project requested from the developer is quoted and charged separately from the license fee** — custom features, changes to existing behavior, bug fixes raised on your schedule, integration, deployment, migration, version upgrades, maintenance, a support commitment or SLA, and training are each a separate engagement, agreed and priced on their own.

Holding a commercial license does not put a request ahead of anyone else's, and does not oblige the developer to take it on.

## Harness core source

Separately negotiable, outside the commercial license above: a **one-time buyout** of the harness core source.

| Item | Detail |
|---|---|
| What is sold | An unbranded harness core — the execution engine on its own |
| What is not included | The Agenvoy product and its trademark, and the `pardnchiu/*` modules it otherwise depends on |
| Version | Fixed at the latest release on the date of sale |
| After delivery | Yours to maintain. No updates, no upstream sync, and no tie to later Agenvoy or `pardnchiu/*` development |
| Fee | A single payment for the source as delivered, with no recurring fee. Development assistance from the developer is quoted separately, on the same terms as above |

Enquire at `dev@pardn.io`.

## Terms

Pricing and terms depend on deployment scale and distribution model, and are agreed per engagement. There is no public price list.

## Contact

Contact the developer directly — [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu), the maintainer and sole copyright holder. Email `dev@pardn.io` with:

- Your organisation
- How Agenvoy would be deployed (hosted service, on-premise, embedded in a distributed product)
- Approximate scale (seats, instances, or end users)
- Any timeline you are working to

## Third-party components

Agenvoy depends on external modules — `go-llm-router`, `go-pkg`, `go-bot`, `go-browser`, `go-scheduler`, `go-sqlkit`, the official MCP go-sdk, and the rest listed in `go.mod` — each under its own license. A commercial license for Agenvoy covers Agenvoy itself; the dependencies keep their own terms.

## Contributions

Contributions are accepted under AGPL-3.0. By opening a pull request you grant the maintainer the right to also distribute your contribution under the commercial license, so that the dual-licensing above stays available.

---

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)
