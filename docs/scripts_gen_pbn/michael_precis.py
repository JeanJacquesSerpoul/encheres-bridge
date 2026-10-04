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

def generate_michaels_deal():
    """Génère une donne valide pour une intervention en Bicolore Michaels par Ouest."""
    deck = [(suit, rank) for suit in ['S', 'H', 'D', 'C'] for rank in "AKQJT98765432"]
    
    while True:
        random.shuffle(deck)
        sud = deck[0:13]    # L'ouvreur adverse
        ouest = deck[13:26]  # L'intervenant Michaels (à tester)
        nord = deck[26:39]   # Le répondant adverse
        est = deck[39:52]    # Le partenaire du Michaels
        
        s_h, s_dist = eval_hand(sud)
        w_h, w_dist = eval_hand(ouest)
        
        # 1. Sud doit avoir une ouverture légitime de 1 en couleur (12-14H)
        if not (12 <= s_h <= 14): continue
        if s_dist['S'] >= 5: ouv_suit = 'S'
        elif s_dist['H'] >= 5: ouv_suit = 'H'
        elif s_dist['D'] >= 4: ouv_suit = 'D'
        else: ouv_suit = 'C'
        ouverture_sud = f"1{ouv_suit}"
        
        # 2. Ouest doit avoir la force requise pour un Michaels (6-11H ou 16+H)
        if not (6 <= w_h <= 11 or w_h >= 16): continue
        
        # 3. Validation de la distribution 5-5 selon la couleur d'ouverture
        if ouv_suit in ['C', 'D']:  # Ouverture en mineure -> Michaels montre les deux majeures (5-5)
            if w_dist['S'] < 5 or w_dist['H'] < 5: continue
            interv_ouest = f"2{ouv_suit}"
            desc = "Michaels Cue-Bid (Promet un bicolore 5-5 Cœur + Pique, zone de combat)"
        
        else:  # Ouverture en majeure -> Michaels montre l'autre majeure (5) + une mineure (5)
            autre_maj = 'H' if ouv_suit == 'S' else 'S'
            if w_dist[autre_maj] < 5: continue
            if w_dist['C'] < 5 and w_dist['D'] < 5: continue
            
            interv_ouest = f"2{ouv_suit}"
            mineure_longue = "Trèfle" if w_dist['C'] >= w_dist['D'] else "Carreau"
            desc = f"Michaels Cue-Bid (Promet l'autre majeure ({autre_maj}) 5ème et une mineure 5ème ({mineure_longue}))"
            
        return {
            'S': sud, 'W': ouest, 'N': nord, 'E': est,
            'ouverture_sud': ouverture_sud, 'interv_ouest': interv_ouest, 'desc_ouest': desc, 
            'w_h': w_h, 'ouv_suit': ouv_suit
        }

def create_pbn_file(filename="michaels_precise.pbn", count=50):
    """Génère le fichier au format .PBN avec le nombre de donnes paramétré."""
    with open(filename, "w", encoding="utf-8") as f:
        f.write("% PBN 2.1\n")
        f.write(f"% Génération automatique de {count} donnes pour l'entraînement au Bicolore Michaels Précisé\n\n")
        
        for i in range(1, count + 1):
            deal = generate_michaels_deal()
            pbn_deal = f"S:{format_pbn_player(deal['S'])} {format_pbn_player(deal['W'])} {format_pbn_player(deal['N'])} {format_pbn_player(deal['E'])}"
            
            f.write(f'[Event "Entrainement Bicolore Michaels - Donne {i}"]\n')
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
            # Séquence : Sud ouvre, Ouest intervient en Michaels (Cue-bid), Nord passe...
            f.write(f'{deal["ouverture_sud"]} {deal["interv_ouest"]} Pass\n')
            f.write(f'[Note "Sud avait ouvert de {deal["ouverture_sud"]}. "]\n')
            f.write(f'[Note "Ouest possède {deal["w_h"]}H et intervient à {deal["interv_ouest"]} : {deal["desc_ouest"]}"]\n\n')
            
    print(f"\nFélicitations ! Le fichier '{filename}' contenant {count} donnes a été créé avec succès.")

if __name__ == "__main__":
    try:
        saisie = input("Combien de donnes de Bicolore Michaels souhaitez-vous générer ? (Par défaut : 50) : ")
        if saisie.strip() == "":
            NOMBRE_DE_DONNES = 50
        else:
            NOMBRE_DE_DONNES = int(saisie)
    except ValueError:
        print("Saisie invalide. Génération par défaut de 50 donnes.")
        NOMBRE_DE_DONNES = 50

    # Lancement de la création
    create_pbn_file(filename=f"michaels_{NOMBRE_DE_DONNES}_donnes.pbn", count=NOMBRE_DE_DONNES)
