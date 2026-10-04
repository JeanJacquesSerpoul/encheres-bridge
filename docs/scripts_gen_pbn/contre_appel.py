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
    
    order = "AKQJT98765432"
    for s in suits:
        suits[s].sort(key=lambda r: order.index(r))
        
    return ".".join(["".join(suits[s]) for s in ['S', 'H', 'D', 'C']])

def generate_takeout_double_deal():
    """Génère une donne valide pour un contre d'appel d'Ouest sur une ouverture de 1D de Sud."""
    deck = [(suit, rank) for suit in ['S', 'H', 'D', 'C'] for rank in "AKQJT98765432"]
    
    while True:
        random.shuffle(deck)
        sud = deck[0:13]
        ouest = deck[13:26]
        nord = deck[26:39]
        est = deck[39:52]
        
        s_h, s_dist = eval_hand(sud)
        w_h, w_dist = eval_hand(ouest)
        e_h, e_dist = eval_hand(est)
        
        # 1. Sud doit avoir une ouverture légitime de 1 Carreau (1D)
        if not (12 <= s_h <= 14): continue
        if s_dist['D'] < 4 or s_dist['S'] >= 5 or s_dist['H'] >= 5: continue
        
        # 2. Ouest doit avoir une main de Contre d'appel classique sur les Carreaux
        if not (12 <= w_h <= 16): continue
        if w_dist['D'] > 2: continue # Court à Carreau (0, 1 ou 2 cartes)
        if w_dist['S'] < 3 or w_dist['H'] < 3 or w_dist['C'] < 3: continue # Tolérance / Soutien partout ailleurs
        
        # Nord passe sur le contre pour laisser Est répondre
        
        # 3. Détermination de la meilleure réponse d'Est (le partenaire du contreur)
        # On cherche sa couleur la plus longue parmi les non-nommées (Priorité Majeures S et H, puis C)
        options = {'S': e_dist['S'], 'H': e_dist['H'], 'C': e_dist['C']}
        # Tri par longueur décroissante, priorité Pique puis Coeur en cas d'égalité
        meilleure_couleur = max(options, key=lambda k: (options[k], k == 'S', k == 'H'))
        
        if e_h <= 7:
            # Zone faible : niveau minimum
            rep_est = f"1{meilleure_couleur}" if meilleure_couleur in ['S', 'H'] else f"2{meilleure_couleur}"
            desc_est = f"Faible obligatoire (0-7 HL) dans sa couleur la plus longue ({meilleure_couleur})"
        elif 8 <= e_h <= 10:
            # Zone encourageante : saut d'un palier
            rep_est = f"2{meilleure_couleur}" if meilleure_couleur in ['S', 'H'] else f"3{meilleure_couleur}"
            desc_est = f"Enchère à saut (8-10 HL), propositionnelle pour la manche en {meilleure_couleur}"
        else:
            # Zone forte : Cue-bid de la couleur adverse (Forcing de manche)
            rep_est = "2D"
            desc_est = "Cue-bid à 2♦ (11+ HL), forcing de manche, cherche le meilleur contrat"
            
        return {
            'S': sud, 'W': ouest, 'N': nord, 'E': est,
            'rep_est': rep_est, 'desc_est': desc_est, 'w_h': w_h, 'e_h': e_h
        }

def create_pbn_file(filename="contre_appel.pbn", count=50):
    """Génère le fichier au format .PBN avec le nombre de donnes paramétré."""
    with open(filename, "w", encoding="utf-8") as f:
        f.write("% PBN 2.1\n")
        f.write(f"% Génération automatique de {count} donnes pour l'entraînement au Contre d'appel\n\n")
        
        for i in range(1, count + 1):
            deal = generate_takeout_double_deal()
            pbn_deal = f"S:{format_pbn_player(deal['S'])} {format_pbn_player(deal['W'])} {format_pbn_player(deal['N'])} {format_pbn_player(deal['E'])}"
            
            f.write(f'[Event "Entrainement Contre d\'appel - Donne {i}"]\n')
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
            f.write(f'1D X Pass {deal["rep_est"]} Pass\n') # S:1♦, W:X (Contre), N:Passe, E:Réponse calculée
            f.write(f'[Note "Ouest contre d\'appel avec {deal["w_h"]}H (court Carreau)."]\n')
            f.write(f'[Note "Est possède {deal["e_h"]}H et choisit : {deal["desc_est"]}"]\n\n')
            
    print(f"\nFélicitations ! Le fichier '{filename}' contenant {count} donnes a été créé avec succès.")

if __name__ == "__main__":
    try:
        saisie = input("Combien de donnes sur le Contre d'appel souhaitez-vous générer ? (Par défaut : 50) : ")
        if saisie.strip() == "":
            NOMBRE_DE_DONNES = 50
        else:
            NOMBRE_DE_DONNES = int(saisie)
    except ValueError:
        print("Saisie invalide. Génération par défaut de 50 donnes.")
        NOMBRE_DE_DONNES = 50

    # Lancement de la création
    create_pbn_file(filename=f"contre_appel_{NOMBRE_DE_DONNES}_donnes.pbn", count=NOMBRE_DE_DONNES)
