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

def generate_rubensohl_deal():
    """Génère une donne valide pour une enchère de Rubensohl par Nord après intervention sur 1SA."""
    deck = [(suit, rank) for suit in ['S', 'H', 'D', 'C'] for rank in "AKQJT98765432"]
    
    while True:
        random.shuffle(deck)
        sud = deck[0:13]    # L'ouvreur d'1SA
        ouest = deck[13:26]  # L'intervenant adverse (ici à 2♠)
        nord = deck[26:39]   # Le répondant (qui va utiliser le Rubensohl)
        est = deck[39:52]
        
        s_h, s_dist = eval_hand(sud)
        w_h, w_dist = eval_hand(ouest)
        n_h, n_dist = eval_hand(nord)
        
        # 1. Sud doit avoir une ouverture d'1 Sans-Atout (15-17H régulier)
        if not (15 <= s_h <= 17): continue
        if any(v <= 1 or v >= 6 for v in s_dist.values()): continue
        
        # 2. Ouest fait une intervention naturelle à 2 Piques (2S) (8-11H, couleur 5ème+) pour gêner
        if not (8 <= w_h <= 12): continue
        if w_dist['S'] < 5: continue
        interv_ouest = "2S"
        
        # 3. Nord possède une couleur au moins 5ème autre que Pique et veut utiliser le Rubensohl
        # Il lui faut une force minimum (généralement dès 7-8 points de combat ou forcing de manche)
        if n_h < 7: continue
        
        couleurs_possibles = [k for k, v in n_dist.items() if v >= 5 and k != 'S']
        if not couleurs_possibles: continue
        target_suit = random.choice(couleurs_possibles)
        
        # Détermination de l'enchère de Rubensohl exacte (Principe du transfert par saut)
        if target_suit == 'C':
            rep_nord = "2NT"  # 2SA = Transfert Trèfle
            rectif_sud = "3C"
            desc = "Rubensohl : 2SA est un transfert pour les Trèfles ♣. Sud doit rectifier à 3♣."
        elif target_suit == 'D':
            rep_nord = "3C"   # 3♣ = Transfert Carreau
            rectif_sud = "3D"
            desc = "Rubensohl : 3♣ est un transfert pour les Carreaux ♦. Sud doit rectifier à 3♦."
        elif target_suit == 'H':
            rep_nord = "3D"   # 3♦ = Transfert Cœur
            rectif_sud = "3H"
            desc = "Rubensohl : 3♦ est un transfert pour les Cœurs ♥. Sud doit rectifier à 3♥."
        else:
            continue
            
        return {
            'S': sud, 'W': ouest, 'N': nord, 'E': est,
            'interv_ouest': interv_ouest, 'rep_nord': rep_nord, 'rectif_sud': rectif_sud, 
            'desc_nord': desc, 's_h': s_h, 'n_h': n_h
        }

def create_pbn_file(filename="rubensohl_entrainement.pbn", count=50):
    """Génère le fichier au format .PBN avec le nombre de donnes paramétré."""
    with open(filename, "w", encoding="utf-8") as f:
        f.write("% PBN 2.1\n")
        f.write(f"% Génération automatique de {count} donnes pour l'entraînement au Rubensohl sur 1SA\n\n")
        
        for i in range(1, count + 1):
            deal = generate_rubensohl_deal()
            pbn_deal = f"S:{format_pbn_player(deal['S'])} {format_pbn_player(deal['W'])} {format_pbn_player(deal['N'])} {format_pbn_player(deal['E'])}"
            
            f.write(f'[Event "Entrainement Rubensohl - Donne {i}"]\n')
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
            # Séquence : Sud 1SA, Ouest intervient à 2♠, Nord emploie le Rubensohl, Est Passe, Sud rectifie
            f.write(f'1NT {deal["interv_ouest"]} {deal["rep_nord"]} Pass\n')
            f.write(f'{deal["rectif_sud"]} Pass\n')
            f.write(f'[Note "Sud ouvre d\'1SA ({deal["s_h"]}H) et subit l\'intervention à {deal["interv_ouest"]}. "]\n')
            f.write(f'[Note "Nord possède {deal["n_h"]}H et utilise le Rubensohl : {deal["desc_nord"]}"]\n\n')
            
    print(f"\nFélicitations ! Le fichier '{filename}' contenant {count} donnes a été créé avec succès.")

if __name__ == "__main__":
    try:
        saisie = input("Combien de donnes de Rubensohl souhaitez-vous générer ? (Par défaut : 50) : ")
        if saisie.strip() == "":
            NOMBRE_DE_DONNES = 50
        else:
            NOMBRE_DE_DONNES = int(saisie)
    except ValueError:
        print("Saisie invalide. Génération par défaut de 50 donnes.")
        NOMBRE_DE_DONNES = 50

    # Lancement de la création
    create_pbn_file(filename=f"rubensohl_{NOMBRE_DE_DONNES}_donnes.pbn", count=NOMBRE_DE_DONNES)
