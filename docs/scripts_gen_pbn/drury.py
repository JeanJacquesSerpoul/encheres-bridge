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

def generate_drury_modern_deal():
    """Génère une donne valide pour la variante Drury moderne demandée."""
    deck = [(suit, rank) for suit in ['S', 'H', 'D', 'C'] for rank in "AKQJT98765432"]
    
    while True:
        random.shuffle(deck)
        sud = deck[0:13]    # L'ouvreur en 3ème position
        ouest = deck[13:26]
        nord = deck[26:39]   # Le répondant (qui va faire le Drury)
        est = deck[39:52]
        
        s_h, s_dist = eval_hand(sud)
        w_h, w_dist = eval_hand(ouest)
        n_h, n_dist = eval_hand(nord)
        
        # 1. Sud, Nord et Ouest doivent PASSER d'entrée (Sud ouvre en 3ème position)
        if s_h >= 12 or n_h >= 12 or w_h >= 12: continue
        
        # 2. Sud possède une ouverture en 3ème position en majeure (11-14 HL)
        majeure_sud = random.choice(['S', 'H'])
        s_hl = s_h + (s_dist[majeure_sud] - 4 if s_dist[majeure_sud] > 4 else 0)
        if not (11 <= s_hl <= 14): continue
        if s_dist[majeure_sud] < 5: continue # Majeure 5ème obligatoire
        
        # 3. Nord doit avoir un fit 3ème ou 4ème et une zone de Drury (9-11 HL)
        n_hl = n_h + (n_dist[majeure_sud] - 3 if n_dist[majeure_sud] > 3 else 0)
        if not (9 <= n_hl <= 11): continue
        if n_dist[majeure_sud] < 3: continue # Fit 3ème minimum requis
        
        ouverture_sud = f"1{majeure_sud}"
        
        # 4. Séquence Drury moderne
        if s_hl <= 12:
            # Main faible : Sud répète sa majeure au niveau de 2
            rep_sud = f"2{majeure_sud}"
            desc_sud = f"Répétition de la couleur = Ouverture faible ({s_hl} HL). Nord va passer."
            rep2_nord = "Pass"
            desc_nord = "Passe sur la conclusion faible de l'ouvreur."
        else:
            # Main forte / Espoir de manche : Sud dit 2♦
            rep_sud = "2D"
            desc_sud = f"2♦ ambigu/forcing avec espoir de manche ({s_hl} HL)."
            
            # Ajustement de la redemande de Nord sur 2♦ selon le nombre d'atouts
            if n_dist[majeure_sud] == 3:
                rep2_nord = f"2{majeure_sud}"
                desc_nord = f"Retour à 2 dans la majeure car Nord n'a que 3 atouts."
            else: # 4 atouts ou plus
                rep2_nord = f"3{majeure_sud}"
                desc_nord = f"Saut à 3 dans la majeure car Nord possède un gros fit de {n_dist[majeure_sud]} atouts."
            
        return {
            'S': sud, 'W': ouest, 'N': nord, 'E': est,
            'ouverture_sud': ouverture_sud, 'rep_sud': rep_sud, 'rep2_nord': rep2_nord,
            'desc_sud': desc_sud, 'desc_nord': desc_nord, 's_hl': s_hl, 'n_hl': n_hl, 
            'majeure': majeure_sud, 'atouts_nord': n_dist[majeure_sud]
        }

def create_pbn_file(filename="drury_moderne.pbn", count=50):
    """Génère le fichier au format .PBN avec les paramètres demandés."""
    with open(filename, "w", encoding="utf-8") as f:
        f.write("% PBN 2.1\n")
        f.write(f"% Génération automatique de {count} donnes pour le Drury moderne (2M=faible, 2D=espoir de manche)\n\n")
        
        for i in range(1, count + 1):
            deal = generate_drury_modern_deal()
            pbn_deal = f"S:{format_pbn_player(deal['S'])} {format_pbn_player(deal['W'])} {format_pbn_player(deal['N'])} {format_pbn_player(deal['E'])}"
            
            f.write(f'[Event "Entrainement Drury Moderne - Donne {i}"]\n')
            f.write('[Site "La Baule-Escoublac"]\n')
            f.write('[Date "2026.10.04"]\n')
            f.write(f'[Board "{i}"]\n')
            f.write('[West "-"]\n')
            f.write('[North "-"]\n')
            f.write('[East "-"]\n')
            f.write('[South "-"]\n')
            f.write('[Dealer "N"]\n')
            f.write('[Vulnerable "None"]\n')
            f.write(f'[Deal "{pbn_deal}"]\n')
            f.write('[Scoring "MP"]\n')
            f.write('[Auction "N"]\n')
            
            # Écriture de la séquence selon la force de la main
            if deal['rep2_nord'] == "Pass":
                f.write(f'Pass Pass {deal["ouverture_sud"]} Pass\n')
                f.write(f'2C Pass {deal["rep_sud"]} Pass\n')
                f.write(f'Pass Pass\n')
            else:
                f.write(f'Pass Pass {deal["ouverture_sud"]} Pass\n')
                f.write(f'2C Pass {deal["rep_sud"]} Pass\n')
                f.write(f'{deal["rep2_nord"]} Pass\n')
                
            f.write(f'[Note "Nord fait un Drury (2♣) fitté {deal["majeure"]} avec {deal["n_hl"]} HL et {deal["atouts_nord"]} atouts."]\n')
            f.write(f'[Note "Sud possède {deal["s_hl"]} HL et répond : {deal["desc_sud"]}"]\n')
            if deal['rep2_nord'] != "Pass":
                f.write(f'[Note "Nord réagit sur 2♦ : {deal["desc_nord"]}"]\n')
            f.write('\n')
            
    print(f"\nFélicitations ! Le fichier '{filename}' contenant {count} donnes a été créé avec succès.")

if __name__ == "__main__":
    try:
        saisie = input("Combien de donnes de Drury moderne souhaitez-vous générer ? (Par défaut : 50) : ")
        if saisie.strip() == "":
            NOMBRE_DE_DONNES = 50
        else:
            NOMBRE_DE_DONNES = int(saisie)
    except ValueError:
        print("Saisie invalide. Génération par défaut de 50 donnes.")
        NOMBRE_DE_DONNES = 50

    # Lancement de la création
    create_pbn_file(filename=f"drury_moderne_{NOMBRE_DE_DONNES}_donnes.pbn", count=NOMBRE_DE_DONNES)
