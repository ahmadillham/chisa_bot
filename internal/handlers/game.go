package handlers

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"

	"chisa_bot/internal/services"
	"chisa_bot/pkg/utils"
)

// GameHandler usage:
// .tebakkata
// .tebakibukota
// .tebaknegara
// .nyerah / .skip
// .leaderboard

type GameType int

const (
	GameNone GameType = iota
	GameKata
	GameIbukota
	GameNegara
	GameBenda
	GameBendera
	GameAngka
	GameKuis
)

type BendaData struct {
	Clue   string
	Answer string
	Valid  []string
}

type KuisData struct {
	Question string
	Answer   string
	Valid    []string
}

var kuisList = []KuisData{
	{"Apa nama ibu kota provinsi Jawa Timur?", "Surabaya", []string{"surabaya"}},
	{"Mata uang negara Jepang adalah?", "Yen", []string{"yen"}},
	{"Binatang yang bisa hidup di air dan di darat disebut?", "Amfibi", []string{"amfibi"}},
	{"Siapakah penemu bola lampu pijar?", "Thomas Alva Edison", []string{"thomas alva edison", "thomas edison", "edison"}},
	{"Tanggal 10 November diperingati sebagai hari apa?", "Hari Pahlawan", []string{"hari pahlawan"}},
	{"Gudeg adalah makanan khas dari daerah mana?", "Yogyakarta", []string{"yogyakarta", "jogja", "jogjakarta"}},
	{"Alat untuk mengukur gempa bumi disebut?", "Seismograf", []string{"seismograf"}},
	{"Benua terbesar di dunia adalah?", "Asia", []string{"asia"}},
	{"Negara manakah yang memiliki julukan 'Negeri Tirai Bambu'?", "China", []string{"china", "tiongkok", "rrc"}},
	{"Apa kepanjangan dari singkatan WHO?", "World Health Organization", []string{"world health organization"}},
	{"Siapakah presiden pertama Republik Indonesia?", "Ir. Soekarno", []string{"ir. soekarno", "soekarno", "sukarno"}},
	{"Naskah teks proklamasi diketik oleh siapa?", "Sayuti Melik", []string{"sayuti melik"}},
	{"Kerajaan Hindu tertua di Indonesia adalah?", "Kutai", []string{"kutai", "kutai kartanegara"}},
	{"Di manakah Jenderal Sudirman memimpin perang gerilya?", "Yogyakarta", []string{"yogyakarta", "hutan jawa tengah", "jawa tengah"}},
	{"Siapakah pahlawan wanita dari Jawa Barat yang terkenal dengan julukan 'Ibu'?", "Dewi Sartika", []string{"dewi sartika"}},
	{"Sumpah Pemuda dibacakan pada tanggal?", "28 Oktober 1928", []string{"28 oktober 1928", "28 oktober"}},
	{"Apa nama kapal Portugis yang pertama kali mendarat di Malaka pada tahun 1511?", "Alfonso de Albuquerque", []string{"alfonso de albuquerque", "kapal alfonso de albuquerque"}},
	{"Candi Borobudur merupakan peninggalan agama?", "Buddha", []string{"buddha"}},
	{"Siapakah wakil presiden pertama Indonesia?", "Moh. Hatta", []string{"moh. hatta", "mohammad hatta", "bung hatta"}},
	{"Apa nama organisasi pergerakan nasional pertama di Indonesia?", "Budi Utomo", []string{"budi utomo"}},
	{"Rumus kimia dari air adalah?", "H2O", []string{"h2o"}},
	{"Planet yang paling dekat dengan Matahari adalah?", "Merkurius", []string{"merkurius"}},
	{"Hewan yang memakan daging disebut?", "Karnivora", []string{"karnivora"}},
	{"Gas yang kita hirup saat bernapas adalah?", "Oksigen", []string{"oksigen", "o2"}},
	{"Bagian tumbuhan yang berfungsi menyerap air dari dalam tanah adalah?", "Akar", []string{"akar"}},
	{"Planet terbesar dalam tata surya kita adalah?", "Jupiter", []string{"jupiter"}},
	{"Perubahan wujud benda dari cair menjadi padat disebut?", "Membeku", []string{"membeku"}},
	{"Reptil besar purba yang sudah punah disebut?", "Dinosaurus", []string{"dinosaurus"}},
	{"Indra manusia yang digunakan untuk mengecap rasa adalah?", "Lidah", []string{"lidah"}},
	{"Sumber energi terbesar bagi bumi adalah?", "Matahari", []string{"matahari"}},
	{"Berapakah hasil dari 7 dikali 8?", "56", []string{"56"}},
	{"Bangun datar yang memiliki 3 sisi disebut?", "Segitiga", []string{"segitiga"}},
	{"Akar pangkat dua dari 100 adalah?", "10", []string{"10"}},
	{"Sudut siku-siku besarnya berapa derajat?", "90", []string{"90 derajat", "90"}},
	{"1 jam ditambah 30 menit sama dengan berapa menit?", "90", []string{"90 menit", "90"}},
	{"Berapakah hasil dari 100 dibagi 4?", "25", []string{"25"}},
	{"Bilangan prima terkecil adalah?", "2", []string{"2"}},
	{"1 lusin sama dengan berapa buah?", "12", []string{"12 buah", "12"}},
	{"Jika sekarang pukul 09.00, 3 jam kemudian pukul berapa?", "12.00", []string{"12.00", "12", "jam 12"}},
	{"Bangun ruang yang memiliki alas dan tutup berbentuk lingkaran adalah?", "Tabung", []string{"tabung"}},
	{"Lawan kata (antonim) dari 'Panjang' adalah?", "Pendek", []string{"pendek"}},
	{"Persamaan kata (sinonim) dari 'Pintar' adalah?", "Pandai", []string{"pandai", "cerdas"}},
	{"'Di mana bumi dipijak, di situ langit dijunjung' adalah contoh dari?", "Peribahasa", []string{"peribahasa"}},
	{"Cerita rakyat tentang anak durhaka yang menjadi batu berasal dari Sumatera Barat adalah?", "Malin Kundang", []string{"malin kundang"}},
	{"Huruf kelima dalam abjad adalah?", "E", []string{"e"}},
	{"Tempat untuk meminjam dan membaca buku disebut?", "Perpustakaan", []string{"perpustakaan"}},
	{"Penulis novel 'Laskar Pelangi' adalah?", "Andrea Hirata", []string{"andrea hirata"}},
	{"Kata 'makan' jika diberi awalan 'di-' menjadi?", "Dimakan", []string{"dimakan"}},
	{"Majas yang melebih-lebihkan sesuatu disebut majas?", "Hiperbola", []string{"hiperbola"}},
	{"Bahasa Inggris dari 'Meja' adalah?", "Table", []string{"table"}},
	{"Apa nama ibu kota provinsi Jawa Barat?", "Bandung", []string{"bandung"}},
	{"Samudra terluas di dunia adalah?", "Pasifik", []string{"samudra pasifik", "pasifik"}},
	{"Lagu kebangsaan Indonesia adalah?", "Indonesia Raya", []string{"indonesia raya"}},
	{"Alat musik yang dimainkan dengan cara dipetik, berasal dari Pulau Rote adalah?", "Sasando", []string{"sasando"}},
	{"Pelabuhan utama di Jakarta bernama?", "Tanjung Priok", []string{"tanjung priok"}},
	{"Negara tetangga Indonesia yang berbatasan darat dengan Kalimantan adalah?", "Malaysia", []string{"malaysia"}},
	{"Gunung tertinggi di Pulau Jawa adalah?", "Semeru", []string{"gunung semeru", "semeru"}},
	{"Mata uang negara Amerika Serikat adalah?", "Dolar", []string{"dolar as", "dollar", "dolar"}},
	{"Julukan kota 'Serambi Mekkah' diberikan untuk kota?", "Banda Aceh", []string{"banda aceh", "aceh"}},
	{"Rumah adat dari Sumatera Barat disebut?", "Rumah Gadang", []string{"rumah gadang"}},
	{"Lambang negara Indonesia adalah?", "Garuda", []string{"garuda pancasila", "garuda"}},
	{"Berapa tahun Indonesia dijajah oleh Jepang?", "3.5", []string{"3,5 tahun", "3.5 tahun", "3,5", "3.5", "tiga setengah"}},
	{"Siapakah pencipta lagu Indonesia Raya?", "W.R. Supratman", []string{"w.r. supratman", "wr supratman", "supratman"}},
	{"Tanggal 21 April diperingati sebagai hari?", "Kartini", []string{"hari kartini", "kartini"}},
	{"Semboyan negara Indonesia adalah?", "Bhinneka Tunggal Ika", []string{"bhinneka tunggal ika"}},
	{"Presiden Indonesia yang ke-3 adalah?", "B.J. Habibie", []string{"b.j. habibie", "bj habibie", "habibie"}},
	{"Kerajaan Islam pertama di Indonesia adalah?", "Samudera Pasai", []string{"samudera pasai"}},
	{"Peristiwa penculikan Soekarno-Hatta sebelum proklamasi disebut peristiwa?", "Rengasdengklok", []string{"rengasdengklok"}},
	{"Warna bendera negara kita adalah?", "Merah Putih", []string{"merah putih"}},
	{"UUD 1945 disahkan pada tanggal?", "18 Agustus 1945", []string{"18 agustus 1945", "18 agustus"}},
	{"Hewan yang menyusui anaknya disebut?", "Mamalia", []string{"mamalia"}},
	{"Bagian mata yang berfungsi mengatur banyaknya cahaya yang masuk adalah?", "Pupil", []string{"pupil"}},
	{"Planet yang memiliki cincin tebal dan indah adalah?", "Saturnus", []string{"saturnus"}},
	{"Hewan terkecil (mikroorganisme) yang dapat menyebabkan penyakit disebut?", "Bakteri", []string{"bakteri", "virus"}},
	{"Proses tumbuhan memasak makanannya sendiri dengan bantuan sinar matahari disebut?", "Fotosintesis", []string{"fotosintesis"}},
	{"Jantung manusia berfungsi untuk?", "Memompa Darah", []string{"memompa darah"}},
	{"Zat hijau daun disebut?", "Klorofil", []string{"klorofil"}},
	{"Alat optik untuk melihat benda-benda yang sangat kecil adalah?", "Mikroskop", []string{"mikroskop"}},
	{"Tulang yang melindungi otak adalah?", "Tengkorak", []string{"tengkorak"}},
	{"Satuan untuk mengukur tegangan listrik adalah?", "Volt", []string{"volt"}},
	{"Sudut yang besarnya kurang dari 90 derajat disebut sudut?", "Lancip", []string{"lancip", "sudut lancip"}},
	{"1 kilogram sama dengan berapa gram?", "1000", []string{"1000 gram", "1000"}},
	{"Bangun datar yang keempat sisinya sama panjang disebut?", "Persegi", []string{"persegi"}},
	{"Angka romawi dari 10 adalah?", "X", []string{"x"}},
	{"Hasil dari 9 pangkat 2 adalah?", "81", []string{"81"}},
	{"Jika sebuah lingkaran dibagi dua sama besar, maka setiap bagian disebut?", "Setengah Lingkaran", []string{"setengah lingkaran", "setengah"}},
	{"Alat untuk mengukur panjang adalah?", "Penggaris", []string{"penggaris", "meteran"}},
	{"1 abad sama dengan berapa tahun?", "100", []string{"100 tahun", "100"}},
	{"Hasil dari 50 dikurangi 25 adalah?", "25", []string{"25"}},
	{"Berapa jumlah sisi pada bangun segiempat?", "4", []string{"4"}},
	{"Olahraga yang menggunakan raket dan kok (shuttlecock) adalah?", "Bulu Tangkis", []string{"bulu tangkis", "badminton"}},
	{"Jumlah pemain dalam satu tim sepak bola adalah?", "11", []string{"11 orang", "11"}},
	{"Tari Kecak berasal dari daerah?", "Bali", []string{"bali"}},
	{"Piala dunia sepak bola diadakan setiap berapa tahun sekali?", "4", []string{"4 tahun", "4 tahun sekali", "4"}},
	{"Alat musik Angklung terbuat dari?", "Bambu", []string{"bambu"}},
	{"Batik diakui oleh UNESCO sebagai warisan budaya dari negara?", "Indonesia", []string{"indonesia"}},
	{"Induk organisasi sepak bola seluruh Indonesia adalah?", "PSSI", []string{"pssi"}},
	{"Lagu 'Gundul-Gundul Pacul' berasal dari daerah?", "Jawa Tengah", []string{"jawa tengah"}},
	{"Seni melipat kertas dari Jepang disebut?", "Origami", []string{"origami"}},
	{"Siapakah pembalap F1 pertama dari Indonesia?", "Rio Haryanto", []string{"rio haryanto"}},
}

var bendaList = []BendaData{
	{"Aku punya wajah tapi tak punya mata, punya jarum tapi tak menjahit. Apakah aku?", "Jam", []string{"jam"}},
	{"Aku makin basah saat mengeringkan badanmu. Apakah aku?", "Handuk", []string{"handuk"}},
	{"Aku punya banyak gigi tapi tidak bisa menggigit. Apakah aku?", "Sisir", []string{"sisir"}},
	{"Aku punya leher tapi tak punya kepala. Apakah aku?", "Botol / Baju", []string{"botol", "baju"}},
	{"Aku harus dipecahkan dulu baru bisa digunakan. Apakah aku?", "Telur", []string{"telur"}},
	{"Aku penuh dengan lubang, tapi masih bisa menahan air. Apakah aku?", "Spons", []string{"spons"}},
	{"Aku punya satu mata tapi tidak bisa melihat. Apakah aku?", "Jarum Jahit", []string{"jarum jahit", "jarum"}},
	{"Aku naik saat hujan turun. Apakah aku?", "Payung", []string{"payung"}},
	{"Aku punya kaki empat, tapi tidak bisa berjalan. Apakah aku?", "Meja / Kursi", []string{"meja", "kursi"}},
	{"Aku punya kota, gunung, dan sungai, tapi tidak ada rumah atau air. Apakah aku?", "Peta", []string{"peta"}},
	{"Aku tinggi saat masih muda, dan pendek saat sudah tua. Apakah aku?", "Lilin", []string{"lilin"}},
	{"Aku bisa berkeliling dunia tapi tetap diam di sudut. Apakah aku?", "Perangko", []string{"perangko"}},
	{"Aku punya banyak kunci tapi tidak bisa membuka satu pintu pun. Apakah aku?", "Piano", []string{"piano"}},
	{"Semakin banyak kamu mengambilku, semakin besar yang aku tinggalkan. Apakah aku?", "Lubang / Jejak Kaki", []string{"lubang", "jejak kaki", "jejak"}},
	{"Aku punya tulang belakang (spine), tapi tidak punya tulang lain. Apakah aku?", "Buku", []string{"buku"}},
	{"Aku punya lidah tapi tidak bisa berbicara atau merasakan rasa. Apakah aku?", "Sepatu", []string{"seatu", "sepatu"}}, // Typo fix: sepatu
	{"Aku berjalan naik dan turun tapi tetap di tempat yang sama. Apakah aku?", "Tangga", []string{"tangga"}},
	{"Aku punya jari tapi tidak punya tulang dan daging. Apakah aku?", "Sarung Tangan", []string{"sarung tangan"}},
	{"Aku selalu ada di depanmu tapi tidak bisa kau lihat. Apakah aku?", "Masa Depan", []string{"masa depan"}},
	{"Aku tidur memakai sepatu dan bangun juga memakai sepatu. Apakah aku?", "Kuda", []string{"kuda", "ban mobil", "ban"}},
	{"Jika kamu menyebut namaku, aku akan pecah/hilang. Apakah aku?", "Kesunyian", []string{"kesunyian", "sunyi", "hening", "keheningan"}},
	{"Aku bisa terbang tanpa sayap dan menangis tanpa mata. Apakah aku?", "Awan", []string{"awan"}},
	{"Aku semakin kecil setiap kali aku mandi. Apakah aku?", "Sabun Batang", []string{"sabun", "sabun batang"}},
	{"Aku punya ranjang (bed) tapi tidak pernah tidur, punya mulut tapi tidak bicara. Apakah aku?", "Sungai", []string{"sungai"}},
	{"Aku dibeli untuk makan, tapi aku sendiri tidak pernah dimakan. Apakah aku?", "Piring / Sendok", []string{"piring", "sendok", "garpu"}},
	{"Aku punya kulit tapi bukan hewan, punya mata banyak tapi bukan nanas. Apakah aku?", "Kentang", []string{"kentang"}},
	{"Aku hanya punya satu warna, tapi punya banyak bentuk dan ukuran, aku selalu menempel padamu saat ada cahaya. Apakah aku?", "Bayangan", []string{"bayangan"}},
	{"Aku punya cincin (ring) tapi tidak punya jari. Apakah aku?", "Telepon", []string{"telepon"}},
	{"Orang membuang kulitku dan memakan isiku, tapi jika isiku ditanam aku bisa tumbuh lagi. Apakah aku?", "Jagung", []string{"jagung", "biji-bijian", "biji bijian"}},
	{"Aku masuk kering dan keluar basah, semakin lama aku di dalam semakin kuat rasanya. Apakah aku?", "Kantong Teh", []string{"kantong teh", "teh"}},
}

var benderaList = []struct{ Flag, Country string }{
	{"🇮🇩", "Indonesia"}, {"🇲🇾", "Malaysia"}, {"🇯🇵", "Jepang"}, {"🇰🇷", "Korea Selatan"},
	{"🇺🇸", "Amerika Serikat"}, {"🇬🇧", "Inggris"}, {"🇫🇷", "Prancis"}, {"🇩🇪", "Jerman"},
	{"🇷🇺", "Rusia"}, {"🇨🇳", "China"}, {"🇦🇺", "Australia"}, {"🇹🇭", "Thailand"},
	{"🇻🇳", "Vietnam"}, {"🇸🇬", "Singapura"}, {"🇵🇭", "Filipina"}, {"🇮🇳", "India"},
	{"🇧🇷", "Brazil"}, {"🇦🇷", "Argentina"}, {"🇨🇦", "Kanada"}, {"🇮🇹", "Italia"},
}

type GameSession struct {
	Type          GameType
	Question      string
	Answer        string   // Primary answer for display
	ValidAnswers []string // All acceptable answers (lowercase)
	StartTime     time.Time
}

type GameHandler struct {
	store    *services.GameStore
	sessions map[string]*GameSession // ChatJID -> Session
	mu       sync.Mutex
}

func NewGameHandler(store *services.GameStore) *GameHandler {
	return &GameHandler{
		store:    store,
		sessions: make(map[string]*GameSession),
	}
}

// -- Data Sources --


var words = []string{
	"meja", "kursi", "lemari", "kasur", "bantal", "lampu", "cermin", "pintu",
	"jendela", "lantai", "buku", "pulpen", "pensil", "kertas", "tas", "sepatu",
	"baju", "celana", "jam", "kunci", "piring", "gelas", "sendok", "garpu",
	"pisau", "kompor", "nasi", "roti", "air", "susu", "matahari", "bulan",
	"bintang", "awan", "hujan", "pohon", "bunga", "tanah", "batu", "rumput",
	"kepala", "tangan", "kaki", "mata", "mulut", "rambut", "ayah", "ibu",
	"anak", "guru", "makan", "minum", "tidur", "bangun", "mandi", "duduk",
	"berdiri", "berjalan", "berlari", "melompat", "melihat", "mendengar",
	"berbicara", "berteriak", "berbisik", "mencium", "merasa", "menyentuh",
	"bertanya", "menjawab", "memasak", "mencuci", "menyapu", "mengepel",
	"menyetrika", "membuka", "menutup", "memotong", "mengaduk", "menuang",
	"membaca", "menulis", "menggambar", "menghitung", "bekerja", "membeli",
	"menjual", "membayar", "mencari", "menemukan", "tertawa", "menangis",
	"tersenyum", "marah", "datang", "pergi", "pulang", "menunggu", "memberi",
	"menerima",
}


type CapitalData struct {
	City  string
	Clues []string
}

var capitals = []CapitalData{
	{"Jakarta", []string{"Kota mana yang memiliki ikon Monumen Nasional (Monas)?", "Apa nama ibu kota negara Indonesia?"}},
	{"Paris", []string{"Kota yang terkenal dengan Menara Eiffel dan Museum Louvre?", "Apa nama ibu kota Prancis yang dijuluki Kota Cinta?"}},
	{"Tokyo", []string{"Kota mana yang memiliki penyeberangan jalan tersibuk di Shibuya?", "Apa nama ibu kota Jepang?"}},
	{"London", []string{"Kota tempat Menara jam Big Ben dan Istana Buckingham berada?", "Apa nama ibu kota Inggris?"}},
	{"Washington D.C.", []string{"Kota mana yang memiliki Gedung Putih (White House)?", "Apa nama ibu kota Amerika Serikat (bukan New York)?"}},
	{"Roma", []string{"Kota yang memiliki bangunan bersejarah Colosseum?", "Apa nama ibu kota Italia?"}},
	{"Kuala Lumpur", []string{"Kota yang terkenal dengan Menara Kembar Petronas?", "Apa nama ibu kota Malaysia?"}},
	{"Beijing", []string{"Kota yang memiliki Kota Terlarang (Forbidden City)?", "Apa nama ibu kota Tiongkok?"}},
	{"Seoul", []string{"Kota yang dibelah oleh Sungai Han dan terkenal dengan K-Pop?", "Apa nama ibu kota Korea Selatan?"}},
	{"Moskow", []string{"Kota mana yang memiliki Lapangan Merah dan Kremlin?", "Apa nama ibu kota Rusia?"}},
	{"Amsterdam", []string{"Kota yang terkenal dengan banyak kanal air dan sepeda?", "Apa nama ibu kota Belanda?"}},
	{"Berlin", []string{"Kota yang memiliki Gerbang Brandenburg dan sisa-sisa tembok pemisah?", "Apa nama ibu kota Jerman?"}},
	{"Bangkok", []string{"Kota mana yang memiliki kuil Grand Palace dan Wat Arun?", "Apa nama ibu kota Thailand?"}},
	{"Kairo", []string{"Kota yang terletak dekat dengan Piramida Giza?", "Apa nama ibu kota Mesir?"}},
	{"Madrid", []string{"Kota markas klub sepak bola Real Madrid?", "Apa nama ibu kota Spanyol?"}},
	{"Brasilia", []string{"Kota ini menggantikan Rio de Janeiro sebagai pusat pemerintahan?", "Apa nama ibu kota Brasil yang tata kotanya berbentuk pesawat?"}},
	{"Ankara", []string{"Kota yang bukan Istanbul, tapi pusat pemerintahan Turki?", "Apa nama ibu kota Turki?"}},
	{"Canberra", []string{"Kota yang memiliki Gedung Opera (Opera House) yang ikonik?\nTunggu, itu Sydney, tapi apa ibu kota Australia yang sebenarnya?", "Apa nama ibu kota Australia?"}},
	{"New Delhi", []string{"Kota yang memiliki gerbang India Gate?", "Apa nama ibu kota India?"}},
	{"Riyadh", []string{"Kota mana yang memiliki gedung tertinggi Kingdom Centre?", "Apa nama ibu kota Arab Saudi?"}},
}

type CountryData struct {
	Country string
	Clues   []string
}

var countries = []CountryData{
	{"Amerika Serikat", []string{"Negara mana yang memiliki landmark Patung Liberty?", "Apa negara yang identik dengan industri film Hollywood?"}},
	{"Jerman", []string{"Negara apa yang terkenal dengan Tembok Berlin?", "Negara mana yang menjadi asal mobil BMW dan Mercedes-Benz?"}},
	{"Brasil", []string{"Negara yang identik dengan tarian Samba dan karnaval meriah?", "Apa negara yang memiliki hutan hujan Amazon terluas?"}},
	{"Korea Selatan", []string{"Negara mana yang merupakan asal dari musik K-Pop?", "Apa negara yang terkenal dengan makanan Kimchi?"}},
	{"Thailand", []string{"Apa negara yang memiliki julukan Negeri Gajah Putih?", "Negara yang terkenal dengan kuliner Tom Yum?"}},
	{"Kanada", []string{"Negara yang bendera nasionalnya bergambar daun Maple?", "Apa negara yang memiliki bagian dari air terjun Niagara di sisi utara?"}},
	{"Singapura", []string{"Negara mana yang memiliki ikon patung singa air (Merlion)?", "Apa negara yang terkenal dengan aturan kebersihan yang sangat ketat?"}},
	{"Swiss", []string{"Apa negara yang terkenal sebagai penghasil jam tangan mewah?", "Negara mana yang identik dengan pegunungan Alpen dan cokelat?"}},
	{"India", []string{"Negara mana yang memiliki bangunan indah Taj Mahal?", "Apa negara yang terkenal dengan industri film Bollywood?"}},
	{"Turki", []string{"Negara yang identik dengan makanan Kebab?", "Apa negara yang memiliki kota Istanbul di dua benua?"}},
	{"Inggris", []string{"Apa negara yang memiliki menara jam Big Ben?", "Negara mana yang identik dengan bus tingkat berwarna merah?"}},
	{"Meksiko", []string{"Negara yang terkenal dengan topi Sombrero?", "Apa negara yang identik dengan makanan Taco dan Nachos?"}},
	{"Indonesia", []string{"Negara mana yang memiliki hewan purba Komodo?", "Apa negara yang terkenal dengan Candi Borobudur?"}},
	{"Yunani", []string{"Apa negara yang identik dengan mitologi dewa-dewi seperti Zeus?", "Negara mana yang merupakan tempat lahirnya Olimpiade?"}},
	{"Italia", []string{"Negara yang terkenal dengan menara miring Pisa?", "Apa negara yang identik dengan Colosseum dan Gladiator?"}},
	{"Uni Emirat Arab", []string{"Negara mana yang memiliki gedung tertinggi di dunia (Burj Khalifa)?", "Apa negara yang memiliki pulau buatan berbentuk pohon palem?"}},
	{"Rusia", []string{"Apa negara yang dijuluki Negeri Beruang Merah?", "Negara mana yang merupakan negara terluas di dunia?"}},
	{"Portugal", []string{"Negara tempat asal pemain bola Cristiano Ronaldo?", "Apa negara yang terkenal dengan kue tart telur (Egg Tart)?"}},
	{"Selandia Baru", []string{"Negara yang terkenal dengan burung Kiwi yang tidak bisa terbang?", "Apa negara yang menjadi lokasi syuting film The Lord of the Rings?"}},
	{"Peru", []string{"Apa negara yang memiliki situs kota kuno Machu Picchu di atas gunung?", "Negara yang identik dengan hewan Llama?"}},
}

// -- Handlers --

func (h *GameHandler) startGame(client *whatsmeow.Client, evt *events.Message, gType GameType) {
	h.mu.Lock()
	defer h.mu.Unlock()

	chatJID := evt.Info.Chat.String()
	if _, exists := h.sessions[chatJID]; exists {
		utils.ReplyText(client, evt, "⚠️ Masih ada game yang berjalan! Selesaikan atau .nyerah dulu.")
		return
	}

	var session *GameSession
	rand.Seed(time.Now().UnixNano())

	switch gType {
	case GameKata:
		word := words[rand.Intn(len(words))]
		// Dynamically scramble the word
		runes := []rune(word)
		rand.Shuffle(len(runes), func(i, j int) {
			runes[i], runes[j] = runes[j], runes[i]
		})
		scrambled := string(runes)
		// Ensure it's not same as original (unlikely for reasonable length, but good safety)
		if scrambled == word {
			// Just swap first two if same
			if len(runes) > 1 {
				runes[0], runes[1] = runes[1], runes[0]
				scrambled = string(runes)
			}
		}

		session = &GameSession{
			Type:         GameKata,
			Question:     scrambled,
			Answer:       strings.ToLower(word),
			ValidAnswers: []string{strings.ToLower(word)},
			StartTime:    time.Now(),
		}
		utils.ReplyText(client, evt, fmt.Sprintf("🎮 *Tebak Kata*\n\nSusun kata berikut: *%s*", strings.ToUpper(session.Question)))

	case GameIbukota:
		item := capitals[rand.Intn(len(capitals))]
		clue := item.Clues[rand.Intn(len(item.Clues))]
		session = &GameSession{
			Type:         GameIbukota,
			Question:     clue,
			Answer:       strings.ToLower(item.City),
			ValidAnswers: []string{strings.ToLower(item.City)},
			StartTime:    time.Now(),
		}
		utils.ReplyText(client, evt, fmt.Sprintf("🎮 *Tebak Ibu Kota*\n\n%s", session.Question))

	case GameNegara:
		item := countries[rand.Intn(len(countries))]
		clue := item.Clues[rand.Intn(len(item.Clues))]
		session = &GameSession{
			Type:         GameNegara,
			Question:     clue,
			Answer:       strings.ToLower(item.Country),
			ValidAnswers: []string{strings.ToLower(item.Country)},
			StartTime:    time.Now(),
		}
		utils.ReplyText(client, evt, fmt.Sprintf("🎮 *Tebak Negara*\n\n%s", session.Question))

	case GameBenda:
		item := bendaList[rand.Intn(len(bendaList))]
		session = &GameSession{
			Type:         GameBenda,
			Question:     item.Clue,
			Answer:       item.Answer,
			ValidAnswers: item.Valid,
			StartTime:    time.Now(),
		}
		utils.ReplyText(client, evt, fmt.Sprintf("🎮 *Tebak Benda*\n\n%s", session.Question))

	case GameBendera:
		item := benderaList[rand.Intn(len(benderaList))]
		session = &GameSession{
			Type:         GameBendera,
			Question:     item.Flag,
			Answer:       item.Country,
			ValidAnswers: []string{strings.ToLower(item.Country)},
			StartTime:    time.Now(),
		}
		utils.ReplyText(client, evt, fmt.Sprintf("🎮 *Tebak Bendera*\n\nBendera negara apa ini?\n%s", session.Question))

	case GameAngka:
		target := rand.Intn(100) + 1 // 1-100
		targetStr := fmt.Sprintf("%d", target)
		session = &GameSession{
			Type:         GameAngka,
			Question:     "Tebak angka antara 1 sampai 100!",
			Answer:       targetStr,
			ValidAnswers: []string{targetStr},
			StartTime:    time.Now(),
		}
		utils.ReplyText(client, evt, "🎮 *Tebak Angka*\n\nSilakan tebak angka antara *1 sampai 100*!")

	case GameKuis:
		item := kuisList[rand.Intn(len(kuisList))]
		session = &GameSession{
			Type:         GameKuis,
			Question:     item.Question,
			Answer:       item.Answer,
			ValidAnswers: item.Valid,
			StartTime:    time.Now(),
		}
		utils.ReplyText(client, evt, fmt.Sprintf("🎮 *Kuis Pengetahuan*\n\n%s", session.Question))
	}

	h.sessions[chatJID] = session

	// Start 30s timer (skip for Tebak Angka)
	if gType != GameAngka {
		go func(s *GameSession) {
		time.Sleep(30 * time.Second)
		h.mu.Lock()
		defer h.mu.Unlock()

		// Check if session still exists and is the same instance
		if current, exists := h.sessions[chatJID]; exists && current == s {
			delete(h.sessions, chatJID)
			utils.ReplyText(client, evt, fmt.Sprintf("⏳ Waktu habis! Jawabannya adalah: *%s*", strings.Title(s.Answer)))
		}
	}(session)
	}
}

func (h *GameHandler) HandleTebakKata(client *whatsmeow.Client, evt *events.Message) {
	h.startGame(client, evt, GameKata)
}

func (h *GameHandler) HandleTebakIbuKota(client *whatsmeow.Client, evt *events.Message) {
	h.startGame(client, evt, GameIbukota)
}

func (h *GameHandler) HandleTebakNegara(client *whatsmeow.Client, evt *events.Message) {
	h.startGame(client, evt, GameNegara)
}

func (h *GameHandler) HandleTebakBenda(client *whatsmeow.Client, evt *events.Message) {
	h.startGame(client, evt, GameBenda)
}

func (h *GameHandler) HandleTebakBendera(client *whatsmeow.Client, evt *events.Message) {
	h.startGame(client, evt, GameBendera)
}

func (h *GameHandler) HandleTebakAngka(client *whatsmeow.Client, evt *events.Message) {
	h.startGame(client, evt, GameAngka)
}

func (h *GameHandler) HandleTebakKuis(client *whatsmeow.Client, evt *events.Message) {
	h.startGame(client, evt, GameKuis)
}

func (h *GameHandler) HandleNyerah(client *whatsmeow.Client, evt *events.Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	chatJID := evt.Info.Chat.String()
	session, exists := h.sessions[chatJID]
	if !exists {
		utils.ReplyText(client, evt, "⚠️ Tidak ada game yang sedang berjalan.")
		return
	}

	utils.ReplyText(client, evt, fmt.Sprintf("🏳️ Anda menyerah! Jawabannya adalah: *%s*", strings.Title(session.Answer)))
	delete(h.sessions, chatJID)
}

func (h *GameHandler) HandleAnswer(client *whatsmeow.Client, evt *events.Message) bool {
	// Check if game active
	chatJID := evt.Info.Chat.String()
	
	h.mu.Lock()
	session, exists := h.sessions[chatJID]
	h.mu.Unlock()

	if !exists {
		return false
	}

	text := strings.ToLower(strings.TrimSpace(utils.GetTextFromMessage(evt)))
	
	isCorrect := false
	for _, valid := range session.ValidAnswers {
		if text == valid {
			isCorrect = true
			break
		}
	}

	// Special handling for Number Guessing to give hints
	if session.Type == GameAngka {
		guess, err := strconv.Atoi(text)
		if err == nil {
			target, _ := strconv.Atoi(session.Answer)
			if guess < target {
				utils.ReplyText(client, evt, "📉 Terlalu kecil! Coba angka lebih besar.")
				return false
			} else if guess > target {
				utils.ReplyText(client, evt, "📈 Terlalu besar! Coba angka lebih kecil.")
				return false
			}
		}
	}

	if isCorrect {
		h.mu.Lock()
		delete(h.sessions, chatJID)
		h.mu.Unlock()

		// Add score
		userJID := evt.Info.Sender.User // Phone number
		// Check if sender name available? evt.Info.PushName
		senderName := evt.Info.PushName
		if senderName == "" {
			senderName = "+" + userJID
		}

		h.store.AddScore(senderName, 1)

		utils.ReplyTextWithMentions(client, evt, fmt.Sprintf("✅ Benar! @%s mendapat 1 poin. 🎉", userJID), []string{evt.Info.Sender.String()})
		return true
	}

	return false // Not the answer, let other handlers process or ignore
}

func (h *GameHandler) HandleLeaderboard(client *whatsmeow.Client, evt *events.Message) {
	scores := h.store.GetLeaderboard()
	if len(scores) == 0 {
		utils.ReplyText(client, evt, "🏆 Leaderboard masih kosong. Mainkan game dulu!")
		return
	}

	// Sort scores
	type entry struct {
		Name  string
		Score int
	}
	var leaderboard []entry
	for k, v := range scores {
		leaderboard = append(leaderboard, entry{k, v})
	}
	sort.Slice(leaderboard, func(i, j int) bool {
		return leaderboard[i].Score > leaderboard[j].Score
	})

	// Format message
	msg := "🏆 *Global Leaderboard* 🏆\n_(Reset setiap 7 hari)_\n\n"
	for i, e := range leaderboard {
		if i >= 10 { // Top 10 only
			break
		}
		medal := ""
		if i == 0 {
			medal = "🥇"
		} else if i == 1 {
			medal = "🥈"
		} else if i == 2 {
			medal = "🥉"
		} else {
			medal = fmt.Sprintf("%d.", i+1)
		}
		msg += fmt.Sprintf("%s %s: *%d* poin\n", medal, e.Name, e.Score)
	}

	utils.ReplyText(client, evt, msg)
}
