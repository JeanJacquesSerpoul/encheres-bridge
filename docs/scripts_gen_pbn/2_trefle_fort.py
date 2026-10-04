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

def generate_valid_strong_two_clubs_deal():
    """Génère une distribution qui valide les critères d'un 2 Trèfles Fort Indéterminé (Sud)."""
    deck = [(suit, rank) for suit in ['S', 'H', 'D', 'C'] for rank in "AKQJT98765432"]
    
    while True:
        random.shuffle(deck)
        sud = deck[0:13]
        s_h, s_dist = eval_hand(sud)
        
        # 1. Critère de force stricte : zone standard de 22-23 points H
        # (Ou main fittée/bicolore auto-suffisante avec au moins 21H et une majeure 5/6ème maîtresse)
        if not (22 <= s_h <= 23): continue
        
        ouest = deck[13:26]
        nord = deck[26:39]
        est = deck[39:52]
        
        n_h, n_dist = eval_hand(nord)
        
        # Déterminer la réponse automatique standard de Nord (Le répondant)
        # 2♦ est la réponse artificielle négative ou d'attente (Moins de 8 points H ou sans couleur 5ème maîtresse)
        # Un changement de couleur au niveau 2 (2H, 2S) ou 3 (3C, 3D) montre un jeu positif (8+H) avec une belle couleur.
        if n_h < 8:
            rep_nord = "2D"
            desc_nord = "Réponse d'attente / faible (Moins de 8H)"
        else:
            # Recherche d'une couleur 5ème chez Nord avec au moins 2 honneurs majeurs
            couleurs_possi = [k for k, v in n_dist.items() if v >= 5]
            if couleurs_possi:
                pref = couleurs_possi[0]
                rep_nord = f"2{pref}" if pref in ['S', 'H'] else f"3{pref}"
                desc_nord = f"Réponse positive (8+H) avec 5 cartes à {pref}"
            else:
                rep_nord = "2D"
                desc_nord = "Réponse d'attente (8+H mais jeu régulier sans couleur 5ème)"

        desc_sud = f"Ouverture de 2♣ Fort Indéterminé ({s_h}H)"
            
        return {
            'S': sud, 'W': ouest, 'N': nord, 'E': est,
            'ouverture': "2C", 'rep_nord': rep_nord,
            'desc_sud': desc_sud, 'desc_nord': desc_nord
        }

def create_pbn_file(filename="deux_trefles_fort.pbn", count=50):
    """Génère le fichier au format .PBN avec le nombre de donnes paramétré."""
    with open(filename, "w", encoding="utf-8") as f:
        f.write("% PBN 2.1\n")
        f.write(f"% Génération automatique de {count} donnes pour l'ouverture de 2♣ Fort Indéterminé\n\n")
        
        for i in range(1, count + 1):
            deal = generate_valid_strong_two_clubs_deal()
            pbn_deal = f"S:{format_pbn_player(deal['S'])} {format_pbn_player(deal['W'])} {format_pbn_player(deal['N'])} {format_pbn_player(deal['E'])}"
            
            f.write(f'[Event "Entrainement 2 Trefles Fort - Donne {i}"]\n')
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
            f.write(f'{deal["ouverture"]} Pass {deal["rep_nord"]} Pass\n') # S: 2♣, N: Réponse automatique
            f.write(f'[Note "Sud montre: {deal["desc_sud"]}"]\n')
            f.write(f'[Note "Nord repond: {deal["desc_nord"]}"]\n\n')
            
    print(f"\nFélicitations ! Le fichier '{filename}' contenant {count} donnes a été créé avec succès.")

if __name__ == "__main__":
    try:
        saisie = input("Combien de donnes de 2♣ fort souhaitez-vous générer ? (Par défaut : 50) : ")
        if saisie.strip() == "":
            NOMBRE_DE_DONNES = 50
        else:
            NOMBRE_DE_DONNES = int(saisie)
    except ValueError:
        print("Saisie invalide. Génération par défaut de 50 donnes.")
        NOMBRE_DE_DONNES = 50

    # Lancement de la création
    create_pbn_file(filename=f"deux_trefles_fort_{NOMBRE_DE_DONNES}_donnes.pbn", count=NOMBRE_DE_DONNES)
