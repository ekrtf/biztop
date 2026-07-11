# IS (Impôt sur les Sociétés) — Cheat Sheet

Source : [entreprendre.service-public.gouv.fr — fiche F23575](https://entreprendre.service-public.gouv.fr/vosdroits/F23575?lang=fr), consultée le 11/07/2026.
C'est le barème utilisé par BizTop pour estimer le résultat net ; les taux sont configurés dans `rules.yml` (section `corporate_tax`) et appliqués par `internal/domain`.

## Taux

| Tranche de bénéfice | Taux |
|---------------------|------|
| Jusqu'à 42 500 €    | **15 %** (taux réduit, sous conditions) |
| Au-delà de 42 500 € | **25 %** (taux normal) |

## Conditions du taux réduit (15 %)

Les deux conditions doivent être remplies :

1. **CA HT ≤ 10 000 000 €** (ramené à 12 mois si l'exercice est plus court ou plus long).
2. **Capital détenu à 75 % au moins par des personnes physiques** (ou par une société elle-même détenue à 75 % par des personnes physiques). Les titres auto-détenus sont exclus du calcul.

→ Davai (SASU, associé unique personne physique, CA < 10 M€) : **éligible**.

## Assiette

L'IS porte sur les bénéfices réalisés en France au cours de l'exercice annuel. En cas de déficit, pas d'IS ; le déficit est reportable (report en avant ou en arrière).

## Exemple (barème appliqué par l'app)

Bénéfice de 100 000 € : `42 500 × 15 % + 57 500 × 25 % = 6 375 + 14 375 = 20 750 €` d'IS, soit un résultat net de 79 250 €.

## Déclaration et paiement

- Déclaration annuelle de résultats : dans les 3 mois de la clôture (clôture au 31/12 : au plus tard le 2e jour ouvré suivant le 1er mai). Télédéclaration obligatoire (EFI/EDI), régime simplifié (2065 + 2033A-G) ou normal (2065 + 2050-2059).
- Paiement en **5 fois** : 4 acomptes trimestriels (15 mars, 15 juin, 15 septembre, 15 décembre) + solde le 15 du 4e mois suivant la clôture (15 mai pour une clôture au 31/12).
- Pas d'acomptes si l'IS de référence est inférieur à 3 000 € ou pour une société nouvellement créée.

## Assujettissement

- IS de plein droit : sociétés de capitaux (SAS/SASU, SARL, SA…).
- IS sur option : EI, EURL, SNC… (les micro-entrepreneurs ne peuvent pas opter).
- Les petites sociétés peuvent opter temporairement pour l'IR sous 7 conditions (dont < 50 salariés et CA < 10 M€).
