# List of LoongArch instructions

This is unofficial compiled list of currently known LoongArch instructions.
Instruction name and encoding are taken from published instruction set manual
from Loongson, then slightly modified to fix some perceived inconsistencies.

## License

Copyright &copy; 2021-2023 Contributors to this project. All rights reserved.

The instruction encoding data tables are licensed under either [CC-BY 4.0 International][CC-BY-4.0]
or [木兰开放作品许可协议 署名，第 1 版 (Mulan Open Works License Attribution, Version 1)][MulanOWL-BY-1.0].

[CC-BY-4.0]: https://creativecommons.org/licenses/by/4.0/
[MulanOWL-BY-1.0]: https://license.coscl.org.cn/MulanOWLBYv1

The accompanying programs and supporting code are licensed under either
[Apache License 2.0][Apache-2.0], [MIT][MIT]
or [木兰宽松许可证，第 2 版 (Mulan Permissive Software License, Version 2)][MulanPSL-2.0].

[Apache-2.0]: https://spdx.org/licenses/Apache-2.0.html
[MIT]: https://spdx.org/licenses/MIT.html
[MulanPSL-2.0]: https://license.coscl.org.cn/MulanPSL2

The license texts are also vendored in the repo:

- [LICENSE.Apache-2.0](./LICENSE.Apache-2.0)
- [LICENSE.CC-BY-4.0](./LICENSE.CC-BY-4.0)
- [LICENSE.MIT](./LICENSE.MIT)
- [LICENSE.MulanOWL-BY-1.0](./LICENSE.MulanOWL-BY-1.0)
- [LICENSE.MulanPSL-2.0](./LICENSE.MulanPSL-2.0)

<details>
<summary>Reasoning behind the licensing decisions</summary>

NOTE: The author is not a lawyer, so take the analysis below with extra
caution. The author believes much of the reasoning are correct however...

The official LoongArch manuals are released by Loongson with all rights
reserved. So first of all, we believe this project is a fair use of the
original work because:

* The data collected here is consistent with the reality, with all original
  content clearly marked and explained. Hence the data tables as a whole
  pose no risk to compatibility across the ecosystem, and by extension,
  Loongson's legal interests.
* We, and likely any potential consumer of the project, work on and/or use
  project for personal and/or academic purposes. And even if the project is
  eventually being consumed for commercial uses, no Loongson interest is
  damaged because no other resource exists that is simultaneously:
    - not being tied to any specific project, and
    - easy for machine consumption.

Regarding the copyright holder of the project, it's the contributor to this
project, because of [《中华人民共和国著作权法》][prc-copyright-law]第十三条、第十四条;
the Chinese copyright law applies because right now all contributors to
the project are PRC citizens to our knowledge.

[prc-copyright-law]: https://www.gov.cn/guoqing/2021-10/29/content_5647633.htm

As to why attribution is required for the data tables, instead of simply
being public domain: according to 《中华人民共和国著作权法》第十条,
the right of attribution (署名权) is a moral right that cannot be waived.
So even if [CC0-1.0](https://creativecommons.org/publicdomain/zero/1.0/) is
applied to the project, in effect it would largely just become CC-BY, and
with the disadvantage of being harder for people to understand because the
"fallback path" in the license text must be taken.

As to why the Mulan licenses are allowed as alternative license choices:
this is because the author wanted to be extra safe regarding the licensing,
especially in the PRC jurisdiction. After checking the Chinese and English
texts of the Mulan licenses, the author decided there is little risk adding
them to the multiple-licensing choices.

</details>

## Modifications to upstream specs

Some instruction mnemonics are changed, some syntactic sugar are dropped:

* `dbcl` is renamed to `dbgcall`, and `hvcl` to `hypcall`.

  The reason why the shorter names are chosen is unclear, but `syscall` remains
  unchanged so the change actually made the naming inconsistent.

* `b[lt,ge][u]` are renamed to `b[gt,le][u]` with corresponding operand order swapped.

  All other two-register instructions have the operand in `rd, rj` order, except
  these jump instructions. To preserve semantics the names have to be tweaked
  accordingly. Also all other instructions with comparison semantics adopt the
  `gt/le` distinction, such as the boundary checks, so `lt/ge` is not consistent
  either.

* `ertn` is renamed to `eret`.

  Not using `ret` for "return" is hard to understand, personally I think it's
  better to stick with widely accepted naming conventions.

* `invtlb` is renamed to `tlbinv`.

  All TLB manipulation instructions start with `tlb` except this one.

* `csrrd`, `csrwr`, `gcsrrd` and `gcsrwr` are removed.

  The `csrrd` and `csrwr` instructions can, to some degree, be seen as special
  cases or "specializations" of `csrxchg`, so remove these in favor of
  non-overlapping encodings. `gcsrrd` and `gcsrwr` are similar.

* The FCSR operands of`movfcsr2gr` and `movgr2fcsr` are marked unsigned
  immediates instead of registers; the mnemonics are renamed to `fcsrrd` and
  `fcsrwr` as well.

  The FCSR is more like a configuration word than real register in terms of
  expected usage; but the manual and vendor toolchain all treat the slot as
  an integer register, which is obviously wrong. Instruction naming is
  adjusted accordingly (to make the FCSR more resemble CSR).

* Instructions that operate on FCCs have the `cf` part renamed to `fcc` in
  their mnemonics.

  Other parts of the manual, especially the register names, refer to the bank
  of 1-bit FP predicates as `fcc`, but mnemonics have `cf`, which is
  inconsistent.

* Naming symmetry is restored for several LBT instructions.

  The x86 and ARM translation assists are almost always named `x86foo` and
  `armfoo` respectively, but annoyingly some are named `setx86foo` and
  `setarmfoo`.

  (After [the issue was reported](https://sourceware.org/pipermail/binutils/2023-June/128100.html)
  I received private confirmation that the names will not be amended but no
  justification was given; I suppose it was because the `set{x86,arm}foo` ops
  are not touching the actual LBT "arch" state, but I'm not sure, and here the
  adjusted names are kept nevertheless for perfect consistency.)

* `asrt[le,gt].d` have the suffix removed.

  According to the manual, there's no mention of operand width anywhere,
  so remove the suffix unless more information is provided.

* `[ld,st]ptr` are renamed `[ld,st]ox4` (Load/Store; Offset x4 or Indexed by 4).

  The instructions are created with accelerating GOT accesses in mind;
  they got their current semantics because the GOT is an array of pointers,
  and pointers are at least 32-bit aligned, meaning the array offsets must be
  at least multiples of 4.

  However, the operation `rd = *(rj + imm << 2)` itself is completely generic,
  and has nothing to do with pointers. So, rename the instructions to hopefully
  avoid having to include the background material (the paragraph above) to
  every LoongArch learner.

* `ext.w.[bh]` are renamed `sext.[bh]`.

  According to the manual these actually sign-extend to full native width
  (GRLEN), regardless of whether GRLEN is 32 or not. Also the previous name is
  not too informative either -- "ext" could as well be "zero-extend" or
  "extract" too, and that's bound to confuse people.

* `bitrev` instructions are renamed `revbit`.

  Byte/halfword-unit-reversing instructions are `revb` and `revh` respectively,
  so unify the naming.

* `lu32i.d` `lu52i.d` are renamed `cu32i.d` `cu52i.d`.

  `lu12i.w` empties low bits, but `lu32i.d` and `lu52i.d` keeps low bits. So
  change the "load" part to "concatenate/connect" to distinguish.

* `pcaddi` is renamed `pcaddu2i`.

  It left-shifts the immediate by 2 before adding, just like its friends
  `pcaddu12i` and `pcaddu18i`, but the `u2` part is curiously missing. Fix the
  inconsistency.

* `alsl.*` are renamed `sladd.*`.

  It's not clear what full name "alsl" actually stands for, so rename to
  reflect the semantics -- "Shift-Left/ScaLed Add".

* `bytepick.*` are renamed `catpick.*`.

  The original names give a false impression of somehow returning or operating
  on a single byte, while in fact it was only *byte*-indexed. Real operation is
  more like first *concatenating* two suffix-sized values then *picking* one
  suffix-sized value out from middle. Besides, cats inside the manuals are
  certainly nice...

## Instruction format notation used in this repo

The instruction format notation used in the official documentation has several
shortcomings that make it difficult for downstream to consume as is:

* Many format names start with a number, like `2RI16`, making them unsuitable as
  identifiers in most programming languages;
* Variants to these "base" formats are not explicitly documented, forcing
  downstream to come up with ad-hoc names themselves;
* Exact encodings of instructions can differ even with identical assembly syntax,
  making the original notation ambiguous to use.

  For example, while `asrtle` takes two register operands, they are not the
  usual `rd, rj` but `rj, rk`. Is this `2R`, variant of `2R` or a special case
  of `3R`?

Due to these reasons, we use a new notation for describing the various LoongArch
instruction formats. The syntax is described in [ABNF] as follows:

[ABNF]: https://en.wikipedia.org/wiki/Augmented_Backus%E2%80%93Naur_form

```
insn-format  =  "EMPTY"
insn-format  =/ reg-slots
insn-format  =/ imm-slots
insn-format  =/ reg-slots imm-slots

reg-slots    = 1*reg
reg          = int-reg / fp-reg / fcc-reg / scratch-reg / lsx-reg / lasx-reg
int-reg      = "D" / "J" / "K" / "A"
fp-reg       = "F" index
fcc-reg      = "C" index
scratch-reg  = "T" index
lsx-reg      = "V" index
lasx-reg     = "X" index

index-length = index length
index        = "d" / "j" / "k" / "a" / "m" / "n"
length       = 1*DIGIT

imm-slots    = 1*imm
imm          = signedness 1*index-length
signedness   = "S" / "U"

// extensions to the above, for expressing the "manual syntax"
manual-insn-format =  "EMPTY"
manual-insn-format =/ 1*manual-arg

manual-arg   =  reg / manual-imm
manual-imm   =  imm
manual-imm   =/ imm postproc

postproc     = "p" pp-frag
pp-frag      = pp-op 1*DIGIT
pp-op        = "p" / "s"
```

This notation has the following advantages:

* One name exactly specifies one encoding;
* All names are valid for use as identifiers in programming languages;
* Unambiguous to parse even when capitalized;
* Properly distinguishes between superficially similar encodings due to design
  warts, for example:
    - `DJ`, `JK`, `FdJ` and `DFj` (presumably all `2R` in official notation), or
    - `DSj20` and `JSd5k16` which are totally different formats, but nearly the
      same in official notation (non-existent `1RI20` and base format `1RI21`).

The field offsets and sizes for the register operand slots are as follows:

|Register slot|Starting bit index|Field size in bits|Bank|
|-------------|------------------|------------------|----|
|`D`|0|5|Integer|
|`J`|5|5|Integer|
|`K`|10|5|Integer|
|`A`|15|5|Integer|
|`C`|Specified by the index character|3|FP condition code|
|`F`|Specified by the index character|5|FP|
|`T`|Specified by the index character|2|LBT scratch|
|`V`|Specified by the index character|5|LSX / 128-bit SIMD|
|`X`|Specified by the index character|5|LASX / 256-bit SIMD|

The bit index specifiers are as follows:

|Character|Starting bit index|
|---------|------------------|
|`d`|0|
|`j`|5|
|`k`|10|
|`a`|15|
|`m`|16|
|`n`|18|

In some formats, a long immediate is broken into multiple fields. The individual
fields are to be concatenated from left (MSB direction) to right (LSB direction)
to form the effective number.
For example, the format officially known as `1RI21` is `JSd5k16` in our
notation; the `Sd5k16` part means `(int32_t)(((x & 0x1f) << 16) | ((x >> 10) & 0xffff))`.

The format that takes no operands is specially named `EMPTY`.
For all other formats, the operand slots are sorted so that all registers
precede the immediates, and from LSB to MSB in either group.
This means even instructions taking `rj, rd` as presented in official manuals
are `DJ` format, and that `bstrpick.d` is `DJUk6Um6` and never something like
`DJUm6Uk6` or `JDUk6Um6`.

Because index characters can only follow certain leading characters, the whole
instruction format string can be capitalized without becoming ambiguous, despite
the index specifiers sharing the same `djka` characters with register operands.
So for example the `JSd5k16` name above can easily be capitalized into `JSD5K16`
for adherence to certain coding styles.

## Original format or "Manual syntax" of instructions

In the original LoongArch ISA manual, many instructions' asm syntax is not
consistent with the above canonicalized notation, but instead are more-or-less
arbitrary (like with the conditional branches, no reason whatsoever for swapping
`rd` and `rj`), and/or seem to abandon consistency for arguably ergonomics
purposes, most commonly by hiding a shifted immediate operand's such nature,
but there are more.

These original formats are recorded in the optional attribute `orig_fmt`,
and they conform to the `manual-insn-format` ABNF item described above instead.
Most notably, this means register operands no longer always come first, hence
the case-insensitive property is lost for some of the formats; also immediates
may optionally have to be "postprocessed" for proper disassembly output (or
"preprocessed" for assembly), they are marked with a `p` followed by one
postprocess operation.

The possible immediate postprocess operations are listed below:

|Notation|Meaning|
|--------|-------|
|`pNN`|`imm + NN`|
|`sNN`|`imm << NN`|
