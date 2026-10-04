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

def count_suit_honors(hand, suit):
    """Compte le nombre d'honneurs majeurs (A, K, Q, J) dans une couleur donnée."""
    suit_cards = [rank for s, rank in hand if s == suit]
    return sum(1 for r in suit_cards if r in ['A', 'K', 'Q', 'J'])

def format_pbn_player(hand):
    """Formate les cartes d'un joueur selon les spécifications PBN (S.H.D.C)."""
    suits = {'S': [], 'H': [], 'D': [], 'C': []}
    for suit, rank in hand:
        suits[suit].append(rank)
    
    order = "AKQJT98765432"
    for s in suits:
        suits[s].sort(key=lambda r: order.index(r))
        
    return ".".join(["".join(suits[s]) for s in ['S', 'H', 'D', 'C']])

def generate_barrage_level_3_deal():
    """Génère une donne valide pour une ouverture de barrage au niveau de 3 par Sud."""
    deck = [(suit, rank) for suit in ['S', 'H', 'D', 'C'] for rank in "AKQJT98765432"]
    
    while True:
        random.shuffle(deck)
        sud = deck[0:13]    # L'ouvreur au niveau de 3
        ouest = deck[13:26]  # L'adversaire en intervention
        nord = deck[26:39]   # Le répondant du barreur
        est = deck[39:52]    # L'adversaire en réveil
        
        s_h, s_dist = eval_hand(sud)
        n_h, n_dist = eval_hand(nord)
        
        # 1. Zone de points H pour un barrage : 6 à 10 H maximum
        if not (6 <= s_h <= 10): continue
        
        # 2. Recherche d'une couleur strictement 7ème chez Sud
        couleurs_7emes = [k for k, v in s_dist.items() if v == 7]
        if not couleurs_7emes: continue
        barrage_suit = couleurs_7emes[0]
        
        # Éviter une autre couleur trop longue (bicolore interdit dans un barrage classique)
        if any(v >= 4 for k, v in s_dist.items() if k != barrage_suit): continue
        
        # Qualité de la couleur : au moins 1 gros honneur (As, Roi ou Dame) pour la sécurité
        if count_suit_honors(sud, barrage_suit) < 1: continue
        
        # Construction de l'enchère
        ouverture = f"3{barrage_suit}"
        nom_couleur = {'S': 'Piques', 'H': 'Coeurs', 'D': 'Carreaux', 'C': 'Trefles'}[barrage_suit]
        desc_sud = f"Barrage à 3 {nom_couleur} ({s_h}H, couleur 7ème)"
        
        # 3. Comportement du répondant (Nord) :
        # - Passe la grande majorité du temps (jeu faible ou moyen).
        # - Prolonge le barrage s'il a du fit et peu de jeu (Loi des levées totales).
        # - Déclare la manche (4M ou 3SA) s'il a un jeu d'ouverture très fort (15+ H) avec des compléments.
        if n_h >= 15:
            if barrage_suit in ['S', 'H']:
                rep_nord = f"4{barrage_suit}"
                desc_nord = "Nord déclare la manche en majeure avec un jeu fort (15+H) et du fit."
            else:
                rep_nord = "3NT"
                desc_nord = "Nord tente 3SA avec un jeu fort et des arrêts dans les autres couleurs."
        elif n_dist[barrage_suit] >= 3 and n_h <= 7:
            # Prolongation tactique du barrage
            rep_nord = f"4{barrage_suit}" if barrage_suit in ['S', 'H'] else f"4{barrage_suit}"
            if barrage_suit in ['S', 'H']:
                desc_nord = f"Prolongation de barrage à 4{barrage_suit} (Loi des levées totales, fit 10ème)."
            else:
                rep_nord = "Pass"
                desc_nord = "Passe par défaut (les barrages mineurs se prolongent rarement)."
        else:
            rep_nord = "Pass"
            desc_nord = "Passe standard : pas assez de jeu pour imposer une manche."
            
        return {
            'S': sud, 'W': ouest, 'N': nord, 'E': est,
            'ouverture': ouverture, 'rep_nord': rep_nord,
            'desc_sud': desc_sud, 'desc_nord': desc_nord, 's_h': s_h, 'n_h': n_h
        }

def create_pbn_file(filename="barrages_niveau_3.pbn", count=50):
    """Génère le fichier au format .PBN avec le nombre de donnes paramétré."""
    with open(filename, "w", encoding="utf-8") as f:
        f.write("% PBN 2.1\n")
        f.write(f"% Génération automatique de {count} donnes pour l'entraînement aux Barrages au niveau de 3\n\n")
        
        for i in range(1, count + 1):
            deal = generate_barrage_level_3_deal()
            pbn_deal = f"S:{format_pbn_player(deal['S'])} {format_pbn_player(deal['W'])} {format_pbn_player(deal['N'])} {format_pbn_player(deal['E'])}"
            
            f.write(f'[Event "Entrainement Barrages - Donne {i}"]\n')
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
            # Séquence : Sud barre au niveau 3, Ouest Passe, Nord répond calculé, Est Passe
            f.write(f'{deal["ouverture"]} Pass {deal["rep_nord"]} Pass\n')
            f.write(f'[Note "Sud ouvre de barrage : {deal["desc_sud"]}"]\n')
            f.write(f'[Note "Nord possède {deal["n_h"]}H et choisit : {deal["desc_nord"]}"]\n\n')
            
    print(f"\nFélicitations ! Le fichier '{filename}' contenant {count} donnes a été créé avec succès.")

if __name__ == "__main__":
    try:
        saisie = input("Combien de donnes de barrage à 3 souhaitez-vous générer ? (Par défaut : 50) : ")
        if saisie.strip() == "":
            NOMBRE_DE_DONNES = 50
        else:
            NOMBRE_DE_DONNES = int(saisie)
    except ValueError:
        print("Saisie invalide. Génération par défaut de 50 donnes.")
        NOMBRE_DE_DONNES = 50

    # Lancement de la création
    create_pbn_file(filename=f"barrages_niveau3_{NOMBRE_DE_DONNES}_donnes.pbn", count=NOMBRE_DE_DONNES)
