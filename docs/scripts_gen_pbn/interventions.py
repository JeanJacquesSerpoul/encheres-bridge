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

def generate_intervention_deal():
    """Génère une donne valide pour une intervention naturelle d'Ouest au niveau de 1."""
    deck = [(suit, rank) for suit in ['S', 'H', 'D', 'C'] for rank in "AKQJT98765432"]
    
    while True:
        random.shuffle(deck)
        sud = deck[0:13]    # L'ouvreur
        ouest = deck[13:26]  # L'intervenant
        nord = deck[26:39]   # Le répondant de l'ouvreur
        est = deck[39:52]    # Le répondant de l'intervenant
        
        s_h, s_dist = eval_hand(sud)
        w_h, w_dist = eval_hand(ouest)
        e_h, e_dist = eval_hand(est)
        
        # 1. Sud doit avoir une ouverture légitime standard de 1 en mineure (1♣ ou 1♦)
        if not (12 <= s_h <= 14): continue
        if s_dist['D'] < 4 and s_dist['C'] < 4: continue
        if s_dist['S'] >= 5 or s_dist['H'] >= 5: continue
        ouv_sud = "1D" if s_dist['D'] >= s_dist['C'] else "1C"
        
        # 2. Ouest doit avoir une main d'intervention naturelle parfaite au niveau de 1 (8-15 H)
        if not (8 <= w_h <= 15): continue
        
        # Choix d'une couleur d'intervention parmi les couleurs restantes (Majeure de préférence)
        couleurs_possibles = [k for k, v in w_dist.items() if v >= 5 and k != ouv_sud[-1]]
        if not couleurs_possibles: continue
        
        # On trie pour privilégier les majeures (S ou H)
        interv_suit = max(couleurs_possibles, key=lambda k: (k in ['S', 'H'], w_dist[k]))
        
        # Sécurité de la couleur : au moins 2 honneurs majeurs pour intervenir sereinement
        if count_suit_honors(ouest, interv_suit) < 2: continue
        
        # Si l'intervention requiert le niveau de 2 (couleur inférieure à l'ouverture), on passe pour ce script
        # Exemple: Ouverture 1♦, intervention 1♠ (Ok). Ouverture 1♦, intervention 2♣ -> hors sujet pour le niveau de 1
        order_suits = ['C', 'D', 'H', 'S']
        if order_suits.index(interv_suit) < order_suits.index(ouv_sud[-1]):
            # Nécessite un palier de 2, on rejette pour rester strictement sur le niveau de 1
            continue
            
        interv_ouest = f"1{interv_suit}"
        
        # Nord passe pour laisser Est s'exprimer
        
        # 3. Calcul de la réponse d'Est (Le partenaire de l'intervenant)
        # S'il y a un fit (3 cartes minimum), Est évalue sa force :
        if e_dist[interv_suit] >= 3:
            if e_h <= 7:
                rep_est = f"2{interv_suit}"
                desc_est = f"Soutien simple fitté (0-7H) - Simple barrage ou barrage constructif"
            elif 8 <= e_h <= 10:
                rep_est = f"3{interv_suit}"
                desc_est = f"Soutien invitatif (8-10H) - Demande au partenaire s'il est maximum"
            else:
                rep_est = f"4{interv_suit}" if interv_suit in ['S', 'H'] else "3NT"
                desc_est = f"Conclusion à la manche (11+H, fit assuré)"
        else:
            # Pas de fit : avec du jeu (8-10H) et un arrêt dans la couleur de Sud, Est dit 1SA
            if 8 <= e_h <= 10:
                rep_est = "1NT"
                desc_est = "Changement de main à 1SA (8-10H) sans fit, promet un arrêt dans la couleur adverse"
            else:
                rep_est = "Pass"
                desc_est = "Passe par manque de fit et jeu insuffisant"
                
        return {
            'S': sud, 'W': ouest, 'N': nord, 'E': est,
            'ouv_sud': ouv_sud, 'interv_ouest': interv_ouest, 'rep_est': rep_est,
            'desc_est': desc_est, 'w_h': w_h, 'interv_suit': interv_suit
        }

def create_pbn_file(filename="interventions.pbn", count=50):
    """Génère le fichier au format .PBN avec le nombre de donnes paramétré."""
    with open(filename, "w", encoding="utf-8") as f:
        f.write("% PBN 2.1\n")
        f.write(f"% Génération automatique de {count} donnes pour l'entraînement aux Interventions de niveau 1\n\n")
        
        for i in range(1, count + 1):
            deal = generate_intervention_deal()
            pbn_deal = f"S:{format_pbn_player(deal['S'])} {format_pbn_player(deal['W'])} {format_pbn_player(deal['N'])} {format_pbn_player(deal['E'])}"
            
            f.write(f'[Event "Entrainement Interventions - Donne {i}"]\n')
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
            # Séquence : Sud ouvre, Ouest intervient au niveau 1, Nord Passe, Est répond
            f.write(f'{deal["ouv_sud"]} {deal["interv_ouest"]} Pass {deal["rep_est"]} Pass\n')
            f.write(f'[Note "Ouest intervient à {deal["interv_ouest"]} avec {deal["w_h"]}H et une belle couleur."]\n')
            f.write(f'[Note "Est choisit la réponse : {deal["desc_est"]}"]\n\n')
            
    print(f"\nFélicitations ! Le fichier '{filename}' contenant {count} donnes a été créé avec succès.")

if __name__ == "__main__":
    try:
        saisie = input("Combien de donnes sur les interventions de niveau 1 souhaitez-vous générer ? (Par défaut : 50) : ")
        if saisie.strip() == "":
            NOMBRE_DE_DONNES = 50
        else:
            NOMBRE_DE_DONNES = int(saisie)
    except ValueError:
        print("Saisie invalide. Génération par défaut de 50 donnes.")
        NOMBRE_DE_DONNES = 50

    # Lancement de la création
    create_pbn_file(filename=f"interventions_niv1_{NOMBRE_DE_DONNES}_donnes.pbn", count=NOMBRE_DE_DONNES)
