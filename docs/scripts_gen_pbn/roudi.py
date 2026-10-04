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

def generate_valid_roudi_deal():
    """Génère une distribution de 52 cartes qui valide les critères du Roudi (Pique ou Coeur)."""
    deck = [(suit, rank) for suit in ['S', 'H', 'D', 'C'] for rank in "AKQJT98765432"]
    
    while True:
        random.shuffle(deck)
        sud = deck[0:13]
        nord = deck[26:39]
        
        s_h, s_dist = eval_hand(sud)
        n_h, n_dist = eval_hand(nord)
        
        # 1. Critères pour Sud (L'ouvreur) : 12-14 H, régulier, pas de majeure 5ème
        if not (12 <= s_h <= 14): continue
        if s_dist['S'] >= 5 or s_dist['H'] >= 5: continue
        if any(v >= 6 for v in s_dist.values()): continue
        
        # Choix aléatoire de la majeure pour Nord (S = Pique, H = Coeur)
        majeure_nord = random.choice(['S', 'H'])
        
        # 2. Critères pour Nord (Le répondant) : 11-15 H, exactement 5 cartes dans la majeure choisie
        if not (11 <= n_h <= 15): continue
        if n_dist[majeure_nord] != 5: continue
        
        # Éviter que Nord ait une autre majeure plus longue ou égale
        autre_maj = 'H' if majeure_nord == 'S' else 'S'
        if n_dist[autre_maj] >= 5: continue
        
        ouest = deck[13:26]
        est = deck[39:52]
        
        # Déterminer la réponse automatique de l'ouvreur (Sud) selon le Roudi 3 paliers
        if s_dist[majeure_nord] == 3:
            if s_h <= 13:
                rep_roudi, desc = "2H", f"Fit 3 cartes à {majeure_nord}, Minimum (12-13H)"
            else:
                rep_roudi, desc = "2S", f"Fit 3 cartes à {majeure_nord}, Maximum (14H)"
        else:
            rep_roudi, desc = "2D", f"Pas de fit à {majeure_nord} (Exactement 2 cartes)"
            
        # Déterminer l'ouverture mineure de Sud (la plus longue)
        ouverture = "1C" if s_dist['C'] >= s_dist['D'] else "1D"
        enchere_majeure_nord = "1S" if majeure_nord == 'S' else "1H"
            
        return {
            'S': sud, 'W': ouest, 'N': nord, 'E': est,
            'ouverture': ouverture, 'enchere_majeure': enchere_majeure_nord,
            'rep_roudi': rep_roudi, 'desc': desc
        }

def create_pbn_file(filename="roudi_entrainement.pbn", count=50):
    """Génère le fichier au format .PBN avec le nombre de donnes paramétré."""
    with open(filename, "w", encoding="utf-8") as f:
        f.write("% PBN 2.1\n")
        f.write(f"% Génération automatique de {count} donnes pour le ROUDI\n\n")
        
        for i in range(1, count + 1):
            deal = generate_valid_roudi_deal()
            pbn_deal = f"S:{format_pbn_player(deal['S'])} {format_pbn_player(deal['W'])} {format_pbn_player(deal['N'])} {format_pbn_player(deal['E'])}"
            
            f.write(f'[Event "Entrainement Roudi - Donne {i}"]\n')
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
            f.write(f'{deal["ouverture"]} Pass {deal["enchere_majeure"]} Pass\n')
            f.write(f'1NT Pass 2C Pass\n')
            f.write(f'{deal["rep_roudi"]} Pass\n')
            f.write(f'[Note "Sud montre: {deal["desc"]}"]\n\n')
            
    print(f"\nFélicitations ! Le fichier '{filename}' contenant {count} donnes a été créé avec succès.")

if __name__ == "__main__":
    # Vous pouvez changer la valeur par défaut ici, ou utiliser l'invite ci-dessous
    try:
        saisie = input("Combien de donnes souhaitez-vous générer ? (Par défaut : 50) : ")
        if saisie.strip() == "":
            NOMBRE_DE_DONNES = 50
        else:
            NOMBRE_DE_DONNES = int(saisie)
    except ValueError:
        print("Saisie invalide. Génération par défaut de 50 donnes.")
        NOMBRE_DE_DONNES = 50

    # Lancement de la création
    create_pbn_file(filename=f"roudi_{NOMBRE_DE_DONNES}_donnes.pbn", count=NOMBRE_DE_DONNES)
