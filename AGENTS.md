# AGENTS - agent-things

Ce fichier définit les règles pour le repo **agent-things** (CLI Things 3 via AppleScript).

## Démarrage de session

- Toujours considérer cette règle comme prioritaire pour ce dossier.
- À chaque nouvelle session d’interaction qui manipule ce projet, déclencher un backup initial via :
  - `agent-things session-start`
  - ou `agent-things backup --dir backups` selon l’implémentation courante
- Le CLI garde en permanence **au maximum 50 backups** (supprime les plus anciens après création d’un nouveau backup).
- Format des backups exigé : `YYYY-MM-DD:HH-MM-SS`.

## Règle d’accès aux données

- L’agent **n’interagit jamais directement** avec la base de données Things (`.thingsdatabase`).
- Toutes les opérations passent uniquement par des appels **AppleScript** via le CLI.

## Règles de comportement général du CLI

- Utiliser `agent-things` pour toutes les opérations (et non la CLI système non contrôlée).
- Vérifier l’état de santé du CLI avant une action longue :
  - `agent-things version` si disponible
  - `agent-things --help`
- En cas d’échec, rapporter clairement la commande exécutée et l’erreur reçue.
- Éviter les opérations destructrices non idempotentes sans backup.

## Opérations attendues à implémenter / documenter

- Recherche et consultation :
  - Rechercher une tâche : `agent-things task search <query>`
  - Recherche globale : `agent-things search <query>`
  - Consulter les tâches du jour / en cours si supporté.
- Projets :
  - Lister projets
  - Ajouter un projet
  - Mettre à jour / éditer un projet
  - Supprimer un projet
- Domaines :
  - Lister les domaines
  - Ajouter un domaine
  - Éditer un domaine
  - Supprimer un domaine
- Tâches :
  - Ajouter une tâche
  - Éditer une tâche
  - Supprimer une tâche
  - Marquer une tâche réalisée
  - Consulter une tâche (id, nom, statut, échéance, tags, notes, sous-tâches)
  - Gérer les notes associées
  - Gérer les sous-tâches
- Délais / dates :
  - Définir/mettre à jour `deadline` et `due date`
  - Supporter les formats de date cohérents (ISO/locales selon input)
  - Tenir compte du fuseau horaire local
- Tags :
  - Ajouter des tags
  - Retirer des tags
  - Filtrer / rechercher par tags

## Backup via CLI

- Commande de backup côté CLI obligatoire pour changer d’état critique.
- Le backup doit être écrit dans le répertoire `backups/` sous le data dir Things.
- Le fichier de backup doit suivre l’horodatage exact `YYYY-MM-DD:HH-MM-SS`.
- Conserver **au maximum 50** backups les plus récents.

## Convention d’exécution

- Préférer les commandes atomiques et explicites.
- Quand plusieurs opérations dépendent d’un état, exécuter dans l’ordre :
  1. backup
  2. action(s) de lecture/écriture
  3. vérification
- Documenter les IDs renvoyés par Things et les effets attendus.
- Si une commande AppleScript est indisponible sur la machine/CI, expliciter le fallback et ne pas modifier la base manuellement.
