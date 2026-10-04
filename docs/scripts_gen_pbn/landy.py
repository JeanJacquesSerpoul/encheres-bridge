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

def generate_landy_deal():
    """Génère une donne valide pour une intervention Landy à 2♣ sur 1SA."""
    deck = [(suit, rank) for suit in ['S', 'H', 'D', 'C'] for rank in "AKQJT98765432"]
    
    while True:
        random.shuffle(deck)
        sud = deck[0:13]    # L'ouvreur à 1SA
        ouest = deck[13:26]  # L'intervenant Landy (2♣)
        nord = deck[26:39]   # Le répondant du camp 1SA
        est = deck[39:52]    # Le partenaire du Landy (Est)
        
        s_h, s_dist = eval_hand(sud)
        w_h, w_dist = eval_hand(ouest)
        e_h, e_dist = eval_hand(est)
        
        # 1. Sud doit avoir une ouverture légitime d'1 Sans-Atout (15-17H régulier)
        if not (15 <= s_h <= 17): continue
        if any(v <= 1 or v >= 6 for v in s_dist.values()): continue
        
        # 2. Ouest doit posséder un bicolore majeur Landy (5-4, 4-5 ou 5-5) avec 8-14H
        if not (8 <= w_h <= 14): continue
        if ouest_bicolore_ok := (w_dist['S'] >= 4 and w_dist['H'] >= 4 and (w_dist['S'] >= 5 or w_dist['H'] >= 5)):
            pass
        else:
            continue
            
        # Enchère artificielle de Landy
        interv_ouest = "2C"
        
        # Nord passe pour laisser Est répondre au Landy
        
        # 3. Choix de la réponse d'Est (Le partenaire du Landy) :
        # - Si longue égale en majeure -> Priorité à la couleur la plus longue chez Est.
        # - Si préférence claire -> Nomme sa meilleure majeure au niveau de 2 (Faible 0-7) ou niveau de manche.
        # - Si égalité parfaite (ex: 3-3 ou 2-2 en majeure) -> Est choisit généralement les Piques ou les Cœurs selon la force.
        # - 2♦ (2D) est le relais artificiel pour demander à l'intervenant de nommer sa majeure 5ème (si Est a un doute).
        if e_dist['S'] > e_dist['H']:
            rep_est = "2S"
            desc_est = "Préférence pour les Piques (couleur la plus longue chez le répondant)."
        elif e_dist['H'] > e_dist['S']:
            rep_est = "2H"
            desc_est = "Préférence pour les Cœurs (couleur la plus longue chez le répondant)."
        else:
            # Égalité de longueur : Est utilise le relais à 2♦ pour demander la couleur 5ème d'Ouest
            rep_est = "2D"
            desc_est = "Longueur égale dans les deux majeures : relais à 2♦ pour demander la majeure 5ème du partenaire."
            
        return {
            'S': sud, 'W': ouest, 'N': nord, 'E': est,
            'interv_ouest': interv_ouest, 'rep_est': rep_est, 'desc_est': desc_est,
            'w_h': w_h, 'e_h': e_h, 'w_dist_s': w_dist['S'], 'w_dist_h': w_dist['H']
        }

def create_pbn_file(filename="landy_entrainement.pbn", count=50):
    """Génère le fichier au format .PBN avec le nombre de donnes paramétré."""
    with open(filename, "w", encoding="utf-8") as f:
        f.write("% PBN 2.1\n")
        f.write(f"% Génération automatique de {count} donnes pour l'entraînement à la convention Landy sur 1SA\n\n")
        
        for i in range(1, count + 1):
            deal = generate_landy_deal()
            pbn_deal = f"S:{format_pbn_player(deal['S'])} {format_pbn_player(deal['W'])} {format_pbn_player(deal['N'])} {format_pbn_player(deal['E'])}"
            
            f.write(f'[Event "Entrainement Landy - Donne {i}"]\n')
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
            # Séquence : Sud ouvre d'1SA, Ouest intervient à 2♣ (Landy), Nord Passe, Est répond
            f.write(f'1NT {deal["interv_ouest"]} Pass {deal["rep_est"]} Pass\n')
            f.write(f'[Note "Ouest intervient Landy (2♣) avec {deal["w_h"]}H et une distribution {deal["w_dist_s"]}-{deal["w_dist_h"]} en majeures."]\n')
            f.write(f'[Note "Est possède {deal["e_h"]}H et choisit de répondre : {deal["desc_est"]}"]\n\n')
            
    print(f"\nFélicitations ! Le fichier '{filename}' contenant {count} donnes a été créé avec succès.")

if __name__ == "__main__":
    try:
        saisie = input("Combien de donnes sur la convention Landy souhaitez-vous générer ? (Par défaut : 50) : ")
        if saisie.strip() == "":
            NOMBRE_DE_DONNES = 50
        else:
            NOMBRE_DE_DONNES = int(saisie)
    except ValueError:
        print("Saisie invalide. Génération par défaut de 50 donnes.")
        NOMBRE_DE_DONNES = 50

    # Lancement de la création
    create_pbn_file(filename=f"landy_1sa_{NOMBRE_DE_DONNES}_donnes.pbn", count=NOMBRE_DE_DONNES)
