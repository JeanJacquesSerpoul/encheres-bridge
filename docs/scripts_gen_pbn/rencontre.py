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

def generate_michaels_or_encounter_deal():
    """Génère une donne valide pour une Enchère de Rencontre par Est en réponse à l'intervention d'Ouest."""
    deck = [(suit, rank) for suit in ['S', 'H', 'D', 'C'] for rank in "AKQJT98765432"]
    
    while True:
        random.shuffle(deck)
        sud = deck[0:13]    # L'ouvreur adverse
        ouest = deck[13:26]  # L'intervenant (1♠)
        nord = deck[26:39]   # Le répondant adverse (passe)
        est = deck[39:52]    # Le répondant fitté (Enchère de rencontre)
        
        s_h, s_dist = eval_hand(sud)
        w_h, w_dist = eval_hand(ouest)
        e_h, e_dist = eval_hand(est)
        
        # 1. Sud doit avoir une ouverture légitime d'1 Cœur (12-14H standard)
        if not (12 <= s_h <= 14): continue
        if s_dist['H'] < 5: continue
        ouv_sud = "1H"
        
        # 2. Ouest doit avoir une intervention naturelle standard d'1 Pique (1♠) (10-14H)
        if not (10 <= w_h <= 14): continue
        if w_dist['S'] < 5: continue
        interv_ouest = "1S"
        
        # Nord passe pour laisser le champ libre à Est
        
        # 3. Est doit avoir les critères parfaits de l'Enchère de Rencontre :
        # - Un fit de 4 cartes dans les Piques d'Ouest
        if e_dist['S'] != 4: continue
        
        # - Une force de zone intermédiaire/forte en soutien (11-14 HLD, soit ~7-10H avec distribution)
        if not (7 <= e_h <= 11): continue
        
        # - Une belle couleur mineure annexe au moins 5ème (Trèfle ou Carreau)
        if e_dist['C'] < 5 and e_dist['D'] < 5: continue
        
        # Choix de la mineure à sauter
        if e_dist['C'] >= 5 and e_dist['C'] >= e_dist['D']:
            rep_est = "3C" # Enchère de rencontre à 3♣
            desc = "Enchère de rencontre à 3♣ : promet un fit 4ème à Pique, 5 cartes à Trèfle et 11-14 HLD."
        else:
            rep_est = "3D" # Enchère de rencontre à 3♦
            desc = "Enchère de rencontre à 3♦ : promet un fit 4ème à Pique, 5 cartes à Carreau et 11-14 HLD."
            
        return {
            'S': sud, 'W': ouest, 'N': nord, 'E': est,
            'ouv_sud': ouv_sud, 'interv_ouest': interv_ouest, 'rep_est': rep_est,
            'desc_est': desc, 'e_h': e_h, 'w_h': w_h
        }

def create_pbn_file(filename="encheres_rencontre.pbn", count=50):
    """Génère le fichier au format .PBN avec le nombre de donnes paramétré."""
    with open(filename, "w", encoding="utf-8") as f:
        f.write("% PBN 2.1\n")
        f.write(f"% Génération automatique de {count} donnes pour l'entraînement aux Enchères de Rencontre\n\n")
        
        for i in range(1, count + 1):
            deal = generate_michaels_or_encounter_deal()
            pbn_deal = f"S:{format_pbn_player(deal['S'])} {format_pbn_player(deal['W'])} {format_pbn_player(deal['N'])} {format_pbn_player(deal['E'])}"
            
            f.write(f'[Event "Entrainement Enchère de Rencontre - Donne {i}"]\n')
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
            # Séquence : Sud 1♥, Ouest 1♠, Nord Passe, Est produit son Enchère de Rencontre
            f.write(f'{deal["ouv_sud"]} {deal["interv_ouest"]} Pass {deal["rep_est"]} Pass\n')
            f.write(f'[Note "Sud ouvre d\'1♥ et Ouest intervient d\'1♠."]\n')
            f.write(f'[Note "Est possède {deal["e_h"]}H et utilise l\'Enchère de Rencontre : {deal["desc_est"]}"]\n\n')
            
    print(f"\nFélicitations ! Le fichier '{filename}' contenant {count} donnes a été créé avec succès.")

if __name__ == "__main__":
    try:
        saisie = input("Combien de donnes d'Enchères de Rencontre souhaitez-vous générer ? (Par défaut : 50) : ")
        if saisie.strip() == "":
            NOMBRE_DE_DONNES = 50
        else:
            NOMBRE_DE_DONNES = int(saisie)
    except ValueError:
        print("Saisie invalide. Génération par défaut de 50 donnes.")
        NOMBRE_DE_DONNES = 50

    # Lancement de la création
    create_pbn_file(filename=f"rencontres_{NOMBRE_DE_DONNES}_donnes.pbn", count=NOMBRE_DE_DONNES)
