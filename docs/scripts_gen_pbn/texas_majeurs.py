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

def generate_texas_majeur_deal():
    """Génère une donne valide pour un Texas Majeur (Cœur ou Pique) de Nord sur 1SA."""
    deck = [(suit, rank) for suit in ['S', 'H', 'D', 'C'] for rank in "AKQJT98765432"]
    
    while True:
        random.shuffle(deck)
        sud = deck[0:13]    # L'ouvreur d'1SA
        ouest = deck[13:26]
        nord = deck[26:39]   # Le répondant (à tester)
        est = deck[39:52]
        
        s_h, s_dist = eval_hand(sud)
        n_h, n_dist = eval_hand(nord)
        
        # 1. Sud doit avoir une ouverture d'1 Sans-Atout (15-17H régulier)
        if not (15 <= s_h <= 17): continue
        if any(v <= 1 or v >= 6 for v in s_dist.values()): continue
        if s_dist['S'] >= 5 or s_dist['H'] >= 5: continue
        
        # 2. Nord doit avoir les critères du Texas Majeur :
        # Une couleur majeure (Cœur ou Pique) au moins 5ème, toutes forces
        if n_dist['H'] < 5 and n_dist['S'] < 5: continue
        
        # Détermination de la majeure la plus longue (priorité au Pique en cas de 5-5 pour le SEF standard à ce palier)
        if n_dist['S'] >= 5 and n_dist['S'] >= n_dist['H']:
            rep_nord = "2H"  # Enchère de 2♥ = Texas Pique
            rectif_sud = "2S" # Rectification obligatoire de Sud à 2♠
            desc = f"Texas Pique (Promet {n_dist['S']} cartes à Pique). Sud est forcé de rectifier à 2♠."
        else:
            rep_nord = "2D"  # Enchère de 2♦ = Texas Cœur
            rectif_sud = "2H" # Rectification obligatoire de Sud à 2♥
            desc = f"Texas Cœur (Promet {n_dist['H']} cartes à Cœur). Sud est forcé de rectifier à 2♥."
            
        return {
            'S': sud, 'W': ouest, 'N': nord, 'E': est,
            'rep_nord': rep_nord, 'rectif_sud': rectif_sud, 'desc_nord': desc, 
            's_h': s_h, 'n_h': n_h
        }

def create_pbn_file(filename="texas_majeurs_1sa.pbn", count=50):
    """Génère le fichier au format .PBN avec le nombre de donnes paramétré."""
    with open(filename, "w", encoding="utf-8") as f:
        f.write("% PBN 2.1\n")
        f.write(f"% Génération automatique de {count} donnes pour l'entraînement aux Texas Majeurs sur 1SA\n\n")
        
        for i in range(1, count + 1):
            deal = generate_texas_majeur_deal()
            pbn_deal = f"S:{format_pbn_player(deal['S'])} {format_pbn_player(deal['W'])} {format_pbn_player(deal['N'])} {format_pbn_player(deal['E'])}"
            
            f.write(f'[Event "Entrainement Texas Majeur - Donne {i}"]\n')
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
            # Séquence : 1SA - Passe - 2♦/2♥ (Texas) - Passe - Rectification de l'ouvreur
            f.write(f'1NT Pass {deal["rep_nord"]} Pass\n')
            f.write(f'{deal["rectif_sud"]} Pass\n')
            f.write(f'[Note "Sud ouvre d\'1SA ({deal["s_h"]}H)."]\n')
            f.write(f'[Note "Nord possède {deal["n_h"]}H et applique : {deal["desc_nord"]}"]\n\n')
            
    print(f"\nFélicitations ! Le fichier '{filename}' contenant {count} donnes a été créé avec succès.")

if __name__ == "__main__":
    try:
        saisie = input("Combien de donnes de Texas Majeurs souhaitez-vous générer ? (Par défaut : 50) : ")
        if saisie.strip() == "":
            NOMBRE_DE_DONNES = 50
        else:
            NOMBRE_DE_DONNES = int(saisie)
    except ValueError:
        print("Saisie invalide. Génération par défaut de 50 donnes.")
        NOMBRE_DE_DONNES = 50

    # Lancement de la création
    create_pbn_file(filename=f"texas_majeurs_{NOMBRE_DE_DONNES}_donnes.pbn", count=NOMBRE_DE_DONNES)
