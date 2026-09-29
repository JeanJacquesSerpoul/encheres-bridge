# Règles du moteur d'enchères

Les règles ne sont plus écrites dans le code Go. Elles sont réunies dans un fichier de données, que le moteur lit au chargement de la page :

- **[cli/rules/sef_rules.yaml](../cli/rules/sef_rules.yaml)** contient les règles SEF 2024. Elles sont ordonnées : pour chaque enchère, c'est la première règle applicable qui l'emporte. Pour les modifier, voir [cli/rules/README.md](../cli/rules/README.md).
- **[tools/python_tools/SEF_2024_spec.md](../tools/python_tools/SEF_2024_spec.md)** décrit la sémantique exacte de ces règles : motifs de séquence, caractéristiques de la main, langage des conditions.
- **[tools/python_tools/SEF_2024.md](../tools/python_tools/SEF_2024.md)** décrit les conventions du point de vue du bridge.

L'ancienne description, qui détaillait règle par règle le moteur codé en dur, reste dans l'historique git. Elle a été remplacée lors de la réécriture du moteur (branche `experimental/refactoring-moteur`).
