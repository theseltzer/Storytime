DROP TABLE IF EXISTS spots;

CREATE TABLE spots (
    id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    x double precision NOT NULL,
    y double precision NOT NULL,
    radius double precision NOT NULL,
    title_en text NOT NULL,
    body_en text NOT NULL,
    title_tr text NOT NULL,
    body_tr text NOT NULL
);

INSERT INTO spots (x, y, radius, title_en, body_en, title_tr, body_tr) VALUES
(1872, 304, 45,
 'Backend Development',
 'Bioengineer turned backend developer. REST APIs, HTTP clients and servers in Go, with PostgreSQL for persistence — plus Python, SQL and C, and Kotlin on Android. Currently finishing the Boot.dev backend path.',
 'Backend Geliştirme',
 'Biyomühendislikten backend geliştiriciliğine geçiş. Go ile REST API''ler, HTTP istemcileri ve sunucuları; kalıcılık için PostgreSQL — ayrıca Python, SQL, C ve Android tarafında Kotlin. Boot.dev backend eğitimi tamamlanmak üzere.'),

(432, 304, 35,
 'Neurochemistry & Lab Research',
 'Undergraduate researcher at Üsküdar NPFUAM: PCR/qPCR, immunohistochemistry (IHC), histological staining, dissection and sample preparation — following experiments end to end, from tissue handling through imaging to interpretation.',
 'Nörokimya ve Laboratuvar Araştırmaları',
 'Üsküdar NPFUAM''da lisans araştırmacısı: PCR/qPCR, immünohistokimya (IHC), histolojik boyama, diseksiyon ve örnek hazırlama — deneyleri doku hazırlığından görüntüleme ve yoruma kadar baştan sona takip etme.'),

(1168, 304, 50,
 'Bioinformatics & Omics Data',
 'Data scientist in the Omics unit at Acıbadem Labmed: processed and analyzed NGS and whole-genome sequencing (WGS) data, and integrated and benchmarked bioinformatics tools to optimize analysis pipelines.',
 'Biyoenformatik ve Omik Veri',
 'Acıbadem Labmed Omics biriminde veri bilimci: NGS ve tüm genom dizileme (WGS) verilerinin işlenmesi ve analizi; analiz pipeline''larını iyileştirmek için biyoenformatik araçlarının entegrasyonu ve karşılaştırmalı değerlendirmesi.'),

(1872, 1424, 40,
 'Software Projects',
 'Pokedex CLI in Go — a REST client with pagination and an in-memory cache with TTL eviction. A static site generator in Python that parses Markdown into a node tree and renders it as HTML. And the same to-do app built three times in Kotlin — in-memory, then SQLite, then cloud-synced — as deliberate iteration on the data layer.',
 'Yazılım Projeleri',
 'Go ile Pokedex CLI — sayfalama ve TTL tabanlı bellek içi önbellek kullanan bir REST istemcisi. Python ile statik site üreteci — Markdown''ı düğüm ağacına ayrıştırıp HTML olarak işleyen bir araç. Ve Kotlin ile üç kez yazılmış aynı to-do uygulaması: önce bellekte, sonra SQLite, sonra bulut senkronizasyonlu — veri katmanı üzerinde bilinçli bir tekrar.'),

(1392, 784, 35,
 'MTB & Endurance Cycling',
 'Mountain bike racing and long-distance riding — the reason you are steering a bike through this CV instead of clicking a menu.',
 'MTB ve Dayanıklılık Sürüşü',
 'Dağ bisikleti yarışları ve uzun mesafe sürüşleri — bu CV''de menüye tıklamak yerine neden bisiklet sürdüğünüzün sebebi.'),

(432, 880, 40,
 'Education',
 'B.Sc. in Bioengineering, Üsküdar University (2015–2020) — a curriculum that runs life sciences and engineering side by side, which is where both the lab work and the code start. Followed by the MIUUL Data Science & Machine Learning bootcamp.',
 'Eğitim',
 'Üsküdar Üniversitesi Biyomühendislik lisansı (2015–2020) — yaşam bilimleri ile mühendisliği aynı müfredatta yürüten bir bölüm; hem laboratuvar hem yazılım tarafı buradan başlıyor. Ardından MIUUL Veri Bilimi ve Makine Öğrenmesi bootcamp''i.'),

(1936, 880, 35,
 'Languages',
 'Turkish (native), English (fluent), Spanish (B1). Enough Spanish to hold a conversation, not yet enough to win an argument in one.',
 'Diller',
 'Türkçe (ana dil), İngilizce (akıcı), İspanyolca (B1). Sohbet edecek kadar İspanyolca var, tartışmayı kazanacak kadar henüz yok.'),

(464, 1424, 40,
 'Project Leadership — Inconel 718',
 'Project leader on an Inconel 718 superalloy development effort (2021–2022): led a cross-functional team, modelled production parameters statistically, coordinated resources, and kept the engineering and production sides speaking the same language.',
 'Proje Liderliği — Inconel 718',
 'Inconel 718 süperalaşım geliştirme projesinde proje lideri (2021–2022): disiplinler arası bir ekibin yönetimi, üretim parametrelerinin istatistiksel modellenmesi, kaynakların koordinasyonu ve mühendislik ile üretim taraflarının aynı dili konuşmasının sağlanması.');
