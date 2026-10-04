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

def generate_chasse_croise_deal():
    """Génère une donne valide pour un Chassé-Croisé ou un Transfert mineur sur 1SA."""
    deck = [(suit, rank) for suit in ['S', 'H', 'D', 'C'] for rank in "AKQJT98765432"]
    
    while True:
        random.shuffle(deck)
        sud = deck[0:13]    # L'ouvreur d'1SA
        ouest = deck[13:26]
        nord = deck[26:39]   # Le répondant à tester
        est = deck[39:52]
        
        s_h, s_dist = eval_hand(sud)
        n_h, n_dist = eval_hand(nord)
        
        # 1. Sud doit avoir une ouverture d'1 Sans-Atout (15-17H régulier)
        if not (15 <= s_h <= 17): continue
        if any(v <= 1 or v >= 6 for v in s_dist.values()): continue
        if s_dist['S'] >= 5 or s_dist['H'] >= 5: continue
        
        # On choisit aléatoirement si on teste un Chassé-Croisé ou un Texas Mineur
        mode = random.choice(['CHASSE_CROISE', 'TEXAS_MINEUR'])
        
        if mode == 'CHASSE_CROISE':
            # Nord doit avoir au moins 8 points H (espoir de manche)
            if n_h < 8: continue
            
            # Cas 1 : 5 Coeurs et 4 Piques -> Enchère de 2♠
            if n_dist['H'] == 5 and n_dist['S'] == 4:
                rep_nord = "2S"
                desc = "Chassé-Croisé (Promet 5 cartes à Cœur et 4 cartes à Pique, zone de manche)"
            # Cas 2 : 5 Piques et 4 Coeurs -> Enchère de 2SA
            elif n_dist['S'] == 5 and n_dist['H'] == 4:
                rep_nord = "2NT"
                desc = "Chassé-Croisé (Promet 5 cartes à Pique et 4 cartes à Cœur, zone de manche)"
            else:
                continue
                
        else: # TEXAS_MINEUR
            # Nord doit posséder une mineure au moins 6ème
            if n_dist['C'] < 6 and n_dist['D'] < 6: continue
            
            if n_dist['C'] >= 6 and n_dist['C'] >= n_dist['D']:
                rep_nord = "3C"
                desc = "Texas Trèfle (Promet une couleur Trèfle au moins 6ème, demande la rectification à 3♦)"
            elif n_dist['D'] >= 6:
                rep_nord = "3D"
                desc = "Texas Carreau (Promet une couleur Carreau au moins 6ème, demande la rectification à 3♥)"
            else:
                continue
                
        return {
            'S': sud, 'W': ouest, 'N': nord, 'E': est,
            'rep_nord': rep_nord, 'desc_nord': desc, 's_h': s_h, 'n_h': n_h
        }

def create_pbn_file(filename="chasse_croise_1sa.pbn", count=50):
    """Génère le fichier au format .PBN avec le nombre de donnes paramétré."""
    with open(filename, "w", encoding="utf-8") as f:
        f.write("% PBN 2.1\n")
        f.write(f"% Génération automatique de {count} donnes pour le Chassé-Croisé et Texas Mineurs sur 1SA\n\n")
        
        for i in range(1, count + 1):
            deal = generate_chasse_croise_deal()
            pbn_deal = f"S:{format_pbn_player(deal['S'])} {format_pbn_player(deal['W'])} {format_pbn_player(deal['N'])} {format_pbn_player(deal['E'])}"
            
            f.write(f'[Event "Entrainement Chassé-Croisé / Mineures - Donne {i}"]\n')
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
            f.write(f'1NT Pass {deal["rep_nord"]} Pass\n')
            f.write(f'[Note "Sud ouvre d\'1SA ({deal["s_h"]}H)."]\n')
            f.write(f'[Note "Nord possède {deal["n_h"]}H et applique : {deal["desc_nord"]}"]\n\n')
            
    print(f"\nFélicitations ! Le fichier '{filename}' contenant {count} donnes a été créé avec succès.")

if __name__ == "__main__":
    try:
        saisie = input("Combien de donnes (Chassé-Croisé / Texas mineur) souhaitez-vous générer ? (Par défaut : 50) : ")
        if saisie.strip() == "":
            NOMBRE_DE_DONNES = 50
        else:
            NOMBRE_DE_DONNES = int(saisie)
    except ValueError:
        print("Saisie invalide. Génération par défaut de 50 donnes.")
        NOMBRE_DE_DONNES = 50

    # Lancement de la création
    create_pbn_file(filename=f"chasse_croise_1sa_{NOMBRE_DE_DONNES}_donnes.pbn", count=NOMBRE_DE_DONNES)
