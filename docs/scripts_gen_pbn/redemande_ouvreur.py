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

def generate_redemande_ouvreur_deal():
    """Génère une donne où Sud ouvre d'1♦, Nord répond 1♥, et Sud a une redemande précise."""
    deck = [(suit, rank) for suit in ['S', 'H', 'D', 'C'] for rank in "AKQJT98765432"]
    
    while True:
        random.shuffle(deck)
        sud = deck[0:13]    # L'ouvreur à tester
        ouest = deck[13:26]
        nord = deck[26:39]   # Le répondant
        est = deck[39:52]
        
        s_h, s_dist = eval_hand(sud)
        n_h, n_dist = eval_hand(nord)
        
        # 1. Sud doit avoir une ouverture légitime d'1 Carreau (12-19 H, au moins 4 Carreaux)
        if not (12 <= s_h <= 19): continue
        if s_dist['D'] < 4: continue
        if s_dist['S'] >= 5 or s_dist['H'] >= 5: continue # Évite l'ouverture en majeure
        
        # 2. Nord doit avoir une réponse standard d'1 Cœur (6-10 H, au moins 4 Cœurs)
        if not (6 <= n_h <= 10): continue
        if n_dist['H'] < 4: continue
        
        # 3. Calcul de la redemande mathématique et académique de Sud (SEF)
        # Situation A : Sud a un fit 4ème à Cœur
        if s_dist['H'] == 4:
            if 12 <= s_h <= 14:
                red_sud, desc = "2H", "Soutien simple (Fit 4ème, zone minimum 12-14 H)"
            elif 15 <= s_h <= 17:
                red_sud, desc = "3H", "Soutien fort invitatif (Fit 4ème, zone intermédiaire 15-17 H)"
            else:
                red_sud, desc = "4H", "Soutien de manche (Fit 4ème, zone forte 18-19 H)"
                
        # Situation B : Pas de fit, main régulière (distribution 4333, 4432, 5332)
        elif max(s_dist.values()) <= 5 and min(s_dist.values()) >= 2:
            if 12 <= s_h <= 14:
                red_sud, desc = "1NT", "Redemande à 1SA (Main régulière, zone minimum 12-14 H)"
            elif 18 <= s_h <= 19:
                red_sud, desc = "2NT", "Redemande à 2SA (Main régulière, zone forte 18-19 H)"
            else:
                # Zone intermédiaire (15-17) régulière -> aurait dû ouvrir d'1SA ! On rejette.
                continue
                
        # Situation C : Pas de fit, main distribuée (unicolore Carreau ou bicolore)
        else:
            if s_dist['S'] >= 4:
                # Bicolore 4♦ - 4♠ au palier de 1 (Économique)
                red_sud, desc = "1S", "Bicolore économique à 1♠ (Au moins 4 cartes à Pique, 12-19 H)"
            elif s_dist['C'] >= 4 and s_dist['D'] >= 5:
                if s_h >= 16:
                    red_sud, desc = "2C", "Bicolore cher à 2♣ (Au moins 5 Carreaux et 4 Trèfles, zone forte 16+ H)"
                else:
                    red_sud, desc = "2D", "Répétition par défaut à 2♦ (Pas de bicolore cher possible avec moins de 16 H)"
            elif s_dist['D'] >= 6:
                if 12 <= s_h <= 14:
                    red_sud, desc = "2D", "Répétition de la couleur à 2♦ (Unicolore Carreau 6ème, zone minimum 12-14 H)"
                else:
                    red_sud, desc = "3D", "Répétition à saut à 3♦ (Unicolore Carreau 6ème, zone forte 15+ H)"
            else:
                red_sud, desc = "2D", "Répétition par défaut à 2♦ (Pas d'autre option fluide)"
                
        return {
            'S': sud, 'W': ouest, 'N': nord, 'E': est,
            'red_sud': red_sud, 'desc_sud': desc, 's_h': s_h, 'n_h': n_h
        }

def create_pbn_file(filename="redemandes_ouvreur.pbn", count=50):
    """Génère le fichier au format .PBN avec le nombre de donnes paramétré."""
    with open(filename, "w", encoding="utf-8") as f:
        f.write("% PBN 2.1\n")
        f.write(f"% Génération automatique de {count} donnes pour l'entraînement aux redemandes de l'ouvreur\n\n")
        
        for i in range(1, count + 1):
            deal = generate_redemande_ouvreur_deal()
            pbn_deal = f"S:{format_pbn_player(deal['S'])} {format_pbn_player(deal['W'])} {format_pbn_player(deal['N'])} {format_pbn_player(deal['E'])}"
            
            f.write(f'[Event "Entrainement Redemandes Ouvreur - Donne {i}"]\n')
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
            # Séquence : Sud ouvre d'1♦, Ouest Passe, Nord répond 1♥, Est Passe... A Sud de parler !
            f.write(f'1D Pass 1H Pass\n')
            f.write(f'{deal["red_sud"]} Pass\n')
            f.write(f'[Note "Sud possède {deal["s_h"]}H et doit redemander : {deal["desc_sud"]}"]\n')
            f.write(f'[Note "Nord avait répondu dans une zone de {deal["n_h"]}H."]\n\n')
            
    print(f"\nFélicitations ! Le fichier '{filename}' contenant {count} donnes a été créé avec succès.")

if __name__ == "__main__":
    try:
        saisie = input("Combien de donnes sur les redemandes de l'ouvreur souhaitez-vous générer ? (Par défaut : 50) : ")
        if saisie.strip() == "":
            NOMBRE_DE_DONNES = 50
        else:
            NOMBRE_DE_DONNES = int(saisie)
    except ValueError:
        print("Saisie invalide. Génération par défaut de 50 donnes.")
        NOMBRE_DE_DONNES = 50

    # Lancement de la création
    create_pbn_file(filename=f"redemandes_ouvreur_{NOMBRE_DE_DONNES}_donnes.pbn", count=NOMBRE_DE_DONNES)
