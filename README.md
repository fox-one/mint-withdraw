# mint-withdraw
mixin node mint withdraw

## Generate keys and verfiy raw address

use the command to generate keys of signer and payee

```bash
$ ./verifier keys
```

output:

```
Signer
Address {Signer Address}
PrivateSpendKey {Signer Private Spend Key}
PublicSpendKey {Signer Public Spend Key}
PrivateViewKey {Signer Private View Key}
PublicViewKey {Signer Public View Key}

Payee
Address {Payee Address}
PrivateSpendKey {Payee Private Spend Key}
PublicSpendKey {Payee Public Spend Key}
PrivateViewKey {Payee Private View Key}
PublicViewKey {Payee Public View Key}

keystore
{
    "s": "{Signer Private Spend Key}",
    "p": "{Payee Private Spend Key}"
}
```

it also generate keystore content(below the "keystore"), save it to `t.json`

then you can use it to verify the rawaddress:

```bash
$ verifier verify -k t.json -raw {Raw Address}
```

if it says "verified", then it matched.

