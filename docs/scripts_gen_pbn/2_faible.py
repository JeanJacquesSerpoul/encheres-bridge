import random

# Valeur des honneurs au bridge
HCP = {'A': 4, 'K': 3, 'Q': 2, 'J': 1, 'T': 0, '9': 0, '8': 0, '7': 0, '6': 0, '5': 0, '4': 0, '3': 0, '2': 0}

def eval_hand(hand):
    """Calcule les points d'honneurs (H) et la distribution d'une main."""
    distribution = {'S': 0, 'H': 0, 'D': 0, 'C': 0}
    for suit, rank in hand:
        distribution[suit] += 1
    h_points = sum(HCP[rank] for _, rank in hand)
    return h_points, distribution

def format_pbn_player(hand):
    """Formate les cartes d'un joueur selon les spécifications PBN (S.H.D.C)."""
    suits = {'S': [], 'H': [], 'D': [], 'C': []}
    for suit, rank in hand:
        suits[suit].append(rank)
    
    # Tri décroissant des cartes (A, K, Q... 2)
    order = "AKQJT98765432"
    for s in suits:
        suits[s].sort(key=lambda r: order.index(r))
        
    return ".".join(["".join(suits[s]) for s in ['S', 'H', 'D', 'C']])

def generate_valid_two_weak_deal():
    """Génère une distribution de 52 cartes valide pour un 2 faible majeur avec un partenaire de 0-20H."""
    deck = [(suit, rank) for suit in ['S', 'H', 'D', 'C'] for rank in "AKQJT98765432"]
    
    while True:
        random.shuffle(deck)
        sud = deck[0:13]    # L'ouvreur de 2 faible
        ouest = deck[13:26]
        nord = deck[26:39]   # Le répondant (0-20H)
        est = deck[39:52]
        
        s_h, s_dist = eval_hand(sud)
        n_h, n_dist = eval_hand(nord)
        
        # 1. Zone de points H pour un 2 faible chez Sud : 6 à 10 H
        if not (6 <= s_h <= 10): continue
        
        # Choix aléatoire de la majeure d'ouverture (S = Pique, H = Coeur)
        majeure = random.choice(['S', 'H'])
        
        # Critères de distribution de l'ouvreur (strictement 6ème)
        if s_dist[majeure] != 6: continue
        autre_maj = 'H' if majeure == 'S' else 'S'
        if s_dist[autre_maj] >= 4: continue
        if any(v >= 6 for k, v in s_dist.items() if k != majeure): continue
        
        # 2. Zone élargie pour Nord (Le partenaire) : 0 à 20 H
        if not (0 <= n_h <= 20): continue
        
        # 3. Calcul automatique de l'action conseillée de Nord (SEF)
        ouverture = f"2{majeure}"
        nom_majeure = "Piques" if majeure == 'S' else "Coeurs"
        
        if n_h >= 15:
            # Main très forte : Nord cherche la manche ou le chelem, il passe par le relais à 2SA
            rep_nord = "2NT"
            desc_nord = f"2SA (Relais interrogatif) : Main très forte ({n_h}H), ambition de manche ou chelem."
        elif 11 <= n_h <= 14:
            if n_dist[majeure] >= 3:
                rep_nord = f"4{majeure}"
                desc_nord = f"4{majeure} : Conclusion à la manche. Jeu fort ({n_h}H) avec un bon fit."
            else:
                rep_nord = "2NT"
                desc_nord = f"2SA (Relais Ogust) : Jeu limite de manche ({n_h}H) sans certitude de fit, demande la force."
        elif 5 <= n_h <= 10 and n_dist[majeure] >= 3:
            # Prolongation de barrage ou barrage constructif
            rep_nord = f"3{majeure}"
            desc_nord = f"3{majeure} : Prolongation de barrage tactique (Loi des levées totales, au moins 3 atouts, jeu faible)."
        elif n_h <= 5 and n_dist[majeure] >= 4:
            # Très gros fit mais jeu nul -> Barrage maximal immédiat
            rep_nord = f"4{majeure}"
            desc_nord = f"4{majeure} : Barrage pur. Main blanche en points mais fittée 10ème au total."
        else:
            rep_nord = "Pass"
            desc_nord = f"PASSE : Main de type partiel ou faible ({n_h}H), aucun espoir de manche."
            
        desc_sud = f"Ouverture de 2 {nom_majeure} faible ({s_h}H, 6 cartes)"
            
        return {
            'S': sud, 'W': ouest, 'N': nord, 'E': est,
            'ouverture': ouverture, 'rep_nord': rep_nord,
            'desc_sud': desc_sud, 'desc_nord': desc_nord, 'n_h': n_h
        }

def create_pbn_file(filename="deux_faible_ajuste.pbn", count=50):
    """Génère le fichier au format .PBN avec le nombre de donnes paramétré."""
    with open(filename, "w", encoding="utf-8") as f:
        f.write("% PBN 2.1\n")
        f.write(f"% Génération automatique de {count} donnes de 2 faible avec partenaire calibré (0-20H)\n\n")
        
        for i in range(1, count + 1):
            deal = generate_valid_two_weak_deal()
            pbn_deal = f"S:{format_pbn_player(deal['S'])} {format_pbn_player(deal['W'])} {format_pbn_player(deal['N'])} {format_pbn_player(deal['E'])}"
            
            f.write(f'[Event "Entrainement 2 Faible (0-20H) - Donne {i}"]\n')
            f.write('[Date "2026.10.04"]\n')
            f.write(f'[Board "{i}"]\n')
            f.write('[West "-"]\n')
            f.write('[North "-"]\n')
            f.write('[East "-"]\n')
            f.write('[South "-"]\n')
            f.write('[Dealer "S"]\n')
            f.write('[Vulnerable "None"]\n')
            f.write(f'[Deal "{pbn_deal}"]\n')
            f.write('[Scoring "MP"]\n')
            f.write('[Auction "S"]\n')
            # Séquence : Sud ouvre de 2 Faible, Ouest Passe, Nord répond calculé, Est Passe
            f.write(f'{deal["ouverture"]} Pass {deal["rep_nord"]} Pass\n')
            f.write(f'[Note "Sud montre : {deal["desc_sud"]}"]\n')
            f.write(f'[Note "Nord possède {deal["n_h"]}H et choisit : {deal["desc_nord"]}"]\n\n')
            
    print(f"\nFélicitations ! Le fichier '{filename}' contenant {count} donnes a été créé avec succès.")

if __name__ == "__main__":
    try:
        saisie = input("Combien de donnes de 2 faible (Nord: 0-20H) souhaitez-vous générer ? (Par défaut : 50) : ")
        if saisie.strip() == "":
            NOMBRE_DE_DONNES = 50
        else:
            NOMBRE_DE_DONNES = int(saisie)
    except ValueError:
        print("Saisie invalide. Génération par défaut de 50 donnes.")
        NOMBRE_DE_DONNES = 50

    # Lancement de la création
    create_pbn_file(filename=f"deux_faible_0_20h_{NOMBRE_DE_DONNES}_donnes.pbn", count=NOMBRE_DE_DONNES)
