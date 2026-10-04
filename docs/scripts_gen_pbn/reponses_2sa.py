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

def generate_response_2nt_deal():
    """Génère une donne valide où Sud ouvre de 2SA et Nord applique le système de réponses."""
    deck = [(suit, rank) for suit in ['S', 'H', 'D', 'C'] for rank in "AKQJT98765432"]
    
    while True:
        random.shuffle(deck)
        sud = deck[0:13]    # L'ouvreur de 2SA
        ouest = deck[13:26]
        nord = deck[26:39]   # Le répondant à tester
        est = deck[39:52]
        
        s_h, s_dist = eval_hand(sud)
        n_h, n_dist = eval_hand(nord)
        
        # 1. Sud doit avoir une ouverture légitime de 2 Sans-Atout (20-21H régulier)
        if not (20 <= s_h <= 21): continue
        if any(v <= 1 or v >= 6 for v in s_dist.values()): continue # Pas de court ni de couleur 6ème
        if s_dist['S'] >= 5 or s_dist['H'] >= 5: continue # Pas de majeure 5ème au SEF
        
        # 2. Calcul automatique de la meilleure réponse de Nord (SEF)
        # Option A : Texas Majeur (Couleur 5ème+ en majeure, toutes forces)
        if n_dist['H'] >= 5 or n_dist['S'] >= 5:
            if n_dist['H'] >= 5 and n_dist['S'] < n_dist['H']:
                rep_nord, desc = "3D", "Texas Cœur (Promet au moins 5 cartes à Cœur, l'ouvreur doit rectifier à 3♥)"
            else:
                rep_nord, desc = "3H", "Texas Pique (Promet au moins 5 cartes à Pique, l'ouvreur doit rectifier à 3♠)"
                
        # Option B : Stayman 3♣ (Au moins une majeure 4ème et au moins 3-4 points H)
        elif (n_dist['H'] == 4 or n_dist['S'] == 4) and n_h >= 3:
            rep_nord, desc = "3C", "Stayman à 3♣ (Recherche d'un fit 4-4 en majeure, requis dès 3-4H)"
            
        # Option C : Mains régulières ou sans majeure majeure
        else:
            if n_h <= 2:
                rep_nord, desc = "Pass", "Passe (Main faible de 0-2H, aucun espoir de manche face à 20-21H)"
            elif 3 <= n_h <= 10:
                rep_nord, desc = "3NT", "Conclusion à 3SA (Main de manche sans majeure, zone 3-10H)"
            else:
                rep_nord, desc = "4NT", "Quantitatif à 4SA (Espoir de chelem, main régulière de 11-12+ H)"
                
        return {
            'S': sud, 'W': ouest, 'N': nord, 'E': est,
            'rep_nord': rep_nord, 'desc_nord': desc, 's_h': s_h, 'n_h': n_h
        }

def create_pbn_file(filename="reponses_2sa.pbn", count=50):
    """Génère le fichier au format .PBN avec le nombre de donnes paramétré."""
    with open(filename, "w", encoding="utf-8") as f:
        f.write("% PBN 2.1\n")
        f.write(f"% Génération automatique de {count} donnes pour l'entraînement aux réponses sur 2SA\n\n")
        
        for i in range(1, count + 1):
            deal = generate_response_2nt_deal()
            pbn_deal = f"S:{format_pbn_player(deal['S'])} {format_pbn_player(deal['W'])} {format_pbn_player(deal['N'])} {format_pbn_player(deal['E'])}"
            
            f.write(f'[Event "Entrainement Réponses 2SA - Donne {i}"]\n')
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
            # Séquence : Sud ouvre de 2SA, Ouest Passe, Nord produit sa réponse calculée
            f.write(f'2NT Pass {deal["rep_nord"]} Pass\n')
            f.write(f'[Note "Sud ouvre de 2SA avec une main régulière géante de {deal["s_h"]}H."]\n')
            f.write(f'[Note "Nord possède {deal["n_h"]}H et doit répondre : {deal["desc_nord"]}"]\n\n')
            
    print(f"\nFélicitations ! Le fichier '{filename}' contenant {count} donnes a été créé avec succès.")

if __name__ == "__main__":
    try:
        saisie = input("Combien de donnes sur les réponses à 2SA souhaitez-vous générer ? (Par défaut : 50) : ")
        if saisie.strip() == "":
            NOMBRE_DE_DONNES = 50
        else:
            NOMBRE_DE_DONNES = int(saisie)
    except ValueError:
        print("Saisie invalide. Génération par défaut de 50 donnes.")
        NOMBRE_DE_DONNES = 50

    # Lancement de la création
    create_pbn_file(filename=f"reponses_2sa_{NOMBRE_DE_DONNES}_donnes.pbn", count=NOMBRE_DE_DONNES)
