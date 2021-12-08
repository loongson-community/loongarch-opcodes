# List of LoongArch instructions

This is unofficial compiled list of currently known LoongArch instructions.
Instruction name and encoding are taken from published instruction set manual
from Loongson, then slightly modified to fix some perceived inconsistencies.

## Modifications to upstream specs

Some instruction mnemonics are changed, some syntactic sugar are dropped:

* `dbcl` is renamed to `dbgcall`.

  The reason why the shorter name is chosen is unclear, but `syscall` remains
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

* `csrrd` and `csrwr` are removed.

  The two instructions can be seen as special cases of `csrxchg`, so remove
  these for non-overlapping encodings.

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
reg          = int-reg / fp-reg / fcc-reg
int-reg      = "D" / "J" / "K" / "A"
fp-reg       = "F" index
fcc-reg      = "C" index

index-length = index length
index        = "d" / "j" / "k" / "a" / "m"
length       = 1*DIGIT

imm-slots    = 1*imm
imm          = signedness 1*index-length
signedness   = "S" / "U"
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

The bit index specifiers are as follows:

|Character|Starting bit index|
|---------|------------------|
|`d`|0|
|`j`|5|
|`k`|10|
|`a`|15|
|`m`|16|

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
