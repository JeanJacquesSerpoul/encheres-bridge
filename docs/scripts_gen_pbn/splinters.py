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

def generate_splinter_deal():
    """Génère une donne valide pour un Splinter de Nord après une ouverture majeure de Sud."""
    deck = [(suit, rank) for suit in ['S', 'H', 'D', 'C'] for rank in "AKQJT98765432"]
    
    while True:
        random.shuffle(deck)
        sud = deck[0:13]    # L'ouvreur
        ouest = deck[13:26]
        nord = deck[26:39]   # Le répondant (qui va faire le Splinter)
        est = deck[39:52]
        
        s_h, s_dist = eval_hand(sud)
        n_h, n_dist = eval_hand(nord)
        
        # 1. Sud doit avoir une ouverture majeure 5ème standard (12-14 H)
        if not (12 <= s_h <= 14): continue
        if s_dist['S'] < 5 and s_dist['H'] < 5: continue
        
        # On choisit la majeure d'ouverture de Sud (la plus longue)
        ouv_suit = 'S' if s_dist['S'] >= s_dist['H'] else 'H'
        ouverture_sud = f"1{ouv_suit}"
        
        # 2. Nord doit avoir la main idéale pour un Splinter :
        # - Zone de points : 11-13 points H (soit 13-15 points de distribution/Gains avec la courte)
        if not (11 <= n_h <= 13): continue
        
        # - Un gros fit de 4 cartes au moins dans la majeure de Sud
        if n_dist[ouv_suit] < 4: continue
        
        # - Une courte stricte (0 ou 1 carte) dans une couleur annexe
        courtes_possibles = [k for k, v in n_dist.items() if v <= 1 and k != ouv_suit]
        if not courtes_possibles: continue
        splinter_suit = random.choice(courtes_possibles)
        
        # Détermination de l'enchère exacte de Splinter (Saut anormal)
        # Ex: Ouverture 1S (Pique) -> Splinter Coeur à 3H, ou Carreau à 4D, ou Trèfle à 4C
        # Ex: Ouverture 1H (Coeur) -> Splinter Pique à 3S, ou Carreau à 4D, ou Trèfle à 4C
        if splinter_suit == 'S' and ouv_suit == 'H':
            splinter_nord = "3S"
        elif splinter_suit == 'H' and ouv_suit == 'S':
            splinter_nord = "3H"
        else:
            splinter_nord = f"4{splinter_suit}"
            
        # 3. Calcul de la réaction de l'ouvreur (Sud) au Splinter :
        # Si Sud a des points perdus dans la courte de Nord (ex: R-V-x ou D-x-x), sa main perd de sa valeur.
        # S'il a un singleton ou des petites cartes (x-x-x), ses cartes maîtresses ailleurs se marient parfaitement !
        sud_suit_cards = [rank for s, rank in sud if s == splinter_suit]
        points_perdus = sum(HCP[r] for r in sud_suit_cards if r in ['K', 'Q', 'J'])
        
        if points_perdus >= 3:
            red_sud = f"4{ouv_suit}"
            desc_sud = f"Plus-value mauvaise (honneurs perdus à {splinter_suit}). Sud freine à la manche."
        else:
            red_sud = "4NT"
            desc_sud = f"Mariage de mains parfait ! Sud n'a pas de points perdus à {splinter_suit} et lance le Blackwood."
            
        return {
            'S': sud, 'W': ouest, 'N': nord, 'E': est,
            'ouverture_sud': ouverture_sud, 'splinter_nord': splinter_nord, 'red_sud': red_sud,
            'desc_sud': desc_sud, 's_h': s_h, 'n_h': n_h, 'ouv_suit': ouv_suit, 'splinter_suit': splinter_suit
        }

def create_pbn_file(filename="splinters_entrainement.pbn", count=50):
    """Génère le fichier au format .PBN avec le nombre de donnes paramétré."""
    with open(filename, "w", encoding="utf-8") as f:
        f.write("% PBN 2.1\n")
        f.write(f"% Génération automatique de {count} donnes pour l'entraînement à la convention du Splinter\n\n")
        
        for i in range(1, count + 1):
            deal = generate_splinter_deal()
            pbn_deal = f"S:{format_pbn_player(deal['S'])} {format_pbn_player(deal['W'])} {format_pbn_player(deal['N'])} {format_pbn_player(deal['E'])}"
            
            f.write(f'[Event "Entrainement Splinter - Donne {i}"]\n')
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
            # Séquence : Sud ouvre d'1 en majeure, Ouest Passe, Nord fait le Splinter, Est Passe, Sud redemande
            f.write(f'{deal["ouverture_sud"]} Pass {deal["splinter_nord"]} Pass\n')
            f.write(f'{deal["red_sud"]} Pass\n')
            f.write(f'[Note "Nord fait une enchère de Splinter à {deal["splinter_nord"]} (Fit 4eme à {deal["ouv_suit"]} et courte à {deal["splinter_suit"]})."]\n')
            f.write(f'[Note "Sud possède {deal["s_h"]}H et réagit : {deal["desc_sud"]}"]\n\n')
            
    print(f"\nFélicitations ! Le fichier '{filename}' contenant {count} donnes a été créé avec succès.")

if __name__ == "__main__":
    try:
        saisie = input("Combien de donnes de Splinter souhaitez-vous générer ? (Par défaut : 50) : ")
        if saisie.strip() == "":
            NOMBRE_DE_DONNES = 50
        else:
            NOMBRE_DE_DONNES = int(saisie)
    except ValueError:
        print("Saisie invalide. Génération par défaut de 50 donnes.")
        NOMBRE_DE_DONNES = 50

    # Lancement de la création
    create_pbn_file(filename=f"splinters_{NOMBRE_DE_DONNES}_donnes.pbn", count=NOMBRE_DE_DONNES)
