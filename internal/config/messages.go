package config

// Remote messages
const (
	MsgWait           = "⏳ Sedang memproses..."
	MsgError          = "❌ Terjadi kesalahan sistem."
	MsgErrorDownload  = "❌ Gagal mendownload media. Pastikan link publik dan valid."
	MsgErrorUpload    = "❌ Gagal mengirim media."
	MsgInvalidUrl     = "⚠️ Link tidak valid."
	MsgOnlyGroup      = "⚠️ Perintah ini hanya bisa digunakan di dalam grup."
	MsgOnlyAdmin      = "⚠️ Perintah ini hanya untuk admin grup."
	MsgOnlyPrivate    = "⚠️ Perintah ini hanya bisa digunakan di personal chat."
	MsgHelpSticker    = "⚠️ Kirim atau reply gambar/video/GIF dengan caption .sticker atau .s"
	MsgHelpToImg      = "⚠️ Reply sticker dengan caption .toimg"
	MsgHelpShowImg    = "⚠️ Reply pesan View Once dengan caption .toimg"
	MsgWelcome        = "Halo! Saya Chisa Bot. Ketik .menu untuk melihat daftar perintah."
	MsgMenu           = `📋 *Daftar Perintah*
Prefix: . ! /

• .sticker (.s)
• .toimg (Sticker->Img / ViewOnce)
• .dl <link>
• .mp3 <link>
• .tagall
• .warn <tag/reply>
• .resetwarn <tag/reply>
• .kick <member>
• .bansticker (reply sticker)
• .unbansticker (reply sticker)
• .stats
• .menu`
)
