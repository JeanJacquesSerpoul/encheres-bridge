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

def count_rkcb_keys(hand, trump_suit):
    """Compte le nombre de clés RKCB (4 As + le Roi d'atout)."""
    keys = 0
    # Compter les As
    for suit, rank in hand:
        if rank == 'A':
            keys += 1
        # Compter le Roi d'atout
        if suit == trump_suit and rank == 'K':
            keys += 1
    return keys

def has_trump_queen(hand, trump_suit):
    """Vérifie si la main possède la Dame d'atout."""
    return any(s == trump_suit and r == 'Q' for s, r in hand)

def generate_rkcb_deal():
    """Génère une donne valide pour un Blackwood 4SA posé par Sud avec l'atout Pique."""
    deck = [(suit, rank) for suit in ['S', 'H', 'D', 'C'] for rank in "AKQJT98765432"]
    
    while True:
        random.shuffle(deck)
        sud = deck[0:13]    # Le poseur du Blackwood
        ouest = deck[13:26]
        nord = deck[26:39]   # Le répondant au Blackwood
        est = deck[39:52]
        
        s_h, s_dist = eval_hand(sud)
        n_h, n_dist = eval_hand(nord)
        
        # 1. Sud doit avoir un gros jeu de chelem unicolore ou bicolore (19-22 points H/HL)
        if s_h < 19: continue
        if s_dist['S'] < 5: continue # Atout Pique fixé pour l'exercice
        
        # 2. Nord doit avoir un jeu de réponse régulier ou fitté (8-12 H)
        if not (8 <= n_h <= 12): continue
        if n_dist['S'] < 3: continue # Assure le fit figeant les clés
        
        # 3. Calcul de la réponse de Nord au Blackwood 4SA à l'atout Pique (S)
        keys_nord = count_rkcb_keys(nord, 'S')
        queen_nord = has_trump_queen(nord, 'S')
        
        if keys_nord in (1, 4):
            rep_nord = "5C"
            desc_nord = f"5♣ : Promet {keys_nord} clés (Réponse standard de la zone 30-41)"
        elif keys_nord in (0, 3):
            rep_nord = "5D"
            desc_nord = f"5♦ : Promet {keys_nord} clé(s) (Réponse standard de la zone 30-41)"
        elif keys_nord == 2 and not queen_nord:
            rep_nord = "5H"
            desc_nord = "5♥ : Promet exactement 2 clés SANS la Dame d'atout Pique."
        elif keys_nord == 2 and queen_nord:
            rep_nord = "5S"
            desc_nord = "5♠ : Promet exactement 2 clés AVEC la Dame d'atout Pique."
        else:
            continue # Cas d'erreur ou de main impossible
            
        return {
            'S': sud, 'W': ouest, 'N': nord, 'E': est,
            'rep_nord': rep_nord, 'desc_nord': desc_nord,
            's_h': s_h, 'n_h': n_h, 'keys': keys_nord, 'queen': queen_nord
        }

def create_pbn_file(filename="blackwood_rkcb.pbn", count=50):
    """Génère le fichier au format .PBN avec le nombre de donnes paramétré."""
    with open(filename, "w", encoding="utf-8") as f:
        f.write("% PBN 2.1\n")
        f.write(f"% Génération automatique de {count} donnes pour l'entraînement au RKCB 4SA (Variante 30-41)\n\n")
        
        for i in range(1, count + 1):
            deal = generate_rkcb_deal()
            pbn_deal = f"S:{format_pbn_player(deal['S'])} {format_pbn_player(deal['W'])} {format_pbn_player(deal['N'])} {format_pbn_player(deal['E'])}"
            
            f.write(f'[Event "Entrainement RKCB 30-41 - Donne {i}"]\n')
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
            # Séquence type : Le fit Pique est implicite grâce au saut de Sud à 4SA au deuxième tour
            f.write(f'1S Pass 2S Pass\n')
            f.write(f'4NT Pass {deal["rep_nord"]} Pass\n')
            f.write(f'[Note "Sud ({deal["s_h"]}H) pose le Blackwood à 4SA. Atout agréé : Pique."]\n')
            f.write(f'[Note "Nord possède {deal["keys"]} clés et la Dame d\'atout = {deal["queen"]}. Il répond : {deal["desc_nord"]}"]\n\n')
            
    print(f"\nFélicitations ! Le fichier '{filename}' contenant {count} donnes a été créé avec succès.")

if __name__ == "__main__":
    try:
        saisie = input("Combien de donnes de Blackwood RKCB (30-41) souhaitez-vous générer ? (Par défaut : 50) : ")
        if saisie.strip() == "":
            NOMBRE_DE_DONNES = 50
        else:
            NOMBRE_DE_DONNES = int(saisie)
    except ValueError:
        print("Saisie invalide. Génération par défaut de 50 donnes.")
        NOMBRE_DE_DONNES = 50

    # Lancement de la création
    create_pbn_file(filename=f"rkcb_3041_{NOMBRE_DE_DONNES}_donnes.pbn", count=NOMBRE_DE_DONNES)
