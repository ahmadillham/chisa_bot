package handlers

import (
	"context"
	"fmt"
	"hash/fnv"
	"math/rand"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"

	"chisa_bot/pkg/utils"
)

// FunHandler handles fun & local game commands.
type FunHandler struct{}

// NewFunHandler creates a new FunHandler.
func NewFunHandler() *FunHandler {
	return &FunHandler{}
}

// HandleKerangAjaib gives a random answer to a question (Magic Conch Shell).
func (h *FunHandler) HandleKerangAjaib(client *whatsmeow.Client, evt *events.Message, rawArgs string) {
	if strings.TrimSpace(rawArgs) == "" {
		utils.ReplyText(client, evt, "⚠️ Penggunaan: .kerangajaib <pertanyaan>\nContoh: .kerangajaib Apakah aku ganteng?")
		return
	}

	answers := []string{
		"🐚 Ya.", "🐚 Tidak.", "🐚 Mungkin.", "🐚 Coba lagi.",
		"🐚 Tentu saja!", "🐚 Tidak mungkin.", "🐚 Bisa jadi...",
		"🐚 Jelas iya!", "🐚 Hmm, tidak yakin.",
		"🐚 Lebih baik tidak usah tahu.", "🐚 Pasti!",
		"🐚 Kayaknya sih iya.", "🐚 Nggak deh.",
		"🐚 Menurut bintang-bintang... iya!", "🐚 Tanya lagi nanti ya.",
	}

	answer := answers[rand.Intn(len(answers))]
	reply := fmt.Sprintf("🔮 *Kerang Ajaib*\n\n❓ %s\n\n%s", rawArgs, answer)
	utils.ReplyText(client, evt, reply)
}

// HandleCekKhodam gives a random funny "spirit" name based on the user's input.
func (h *FunHandler) HandleCekKhodam(client *whatsmeow.Client, evt *events.Message, rawArgs string) {
	name := strings.TrimSpace(rawArgs)
	if name == "" {
		utils.ReplyText(client, evt, "⚠️ Penggunaan: .cekkhodam <nama>\nContoh: .cekkhodam Budi")
		return
	}

	khodams := []string{
		"Macan Putih", "Ular Cobra Emas", "Kulkas 2 Pintu", "Tutup Botol",
		"Sendal Jepit", "Naga Hitam", "Kipas Angin", "Tikus Got",
		"Harimau Sumatra", "Remote TV", "Garuda Sakti", "Kompor Meleduk",
		"Singa Barong", "Ember Bocor", "Ayam Jago", "Panci Ajaib",
		"Kucing Oren", "Galon Kosong", "Elang Bondol", "Shower Mati",
		"Buaya Putih", "Rice Cooker", "Kuda Terbang", "Obat Nyamuk",
		"Singa Putih", "Setrika Panas", "Rajawali Emas", "Jemuran Basah",
		"Banteng Api", "Sapu Lidi Sakti", "Ikan Cupang", "WiFi Tetangga",
		"Naga Api", "Kresek Hitam", "Phoenix Merah", "Sandal Bolong",
		"Serigala Arktik", "Dispenser Error", "Kumbang Emas", "Helm Ojol",
	}

	hasher := fnv.New32a()
	hasher.Write([]byte(strings.ToLower(name)))
	idx := int(hasher.Sum32()) % len(khodams)

	reply := fmt.Sprintf("🔮 *Cek Khodam*\n\n👤 Nama: %s\n🐉 Khodam: *%s*", name, khodams[idx])
	utils.ReplyText(client, evt, reply)
}

// HandleCekJodoh calculates a compatibility percentage between two names.
func (h *FunHandler) HandleCekJodoh(client *whatsmeow.Client, evt *events.Message, args []string) {
	if len(args) < 2 {
		utils.ReplyText(client, evt, "⚠️ Penggunaan: .cekjodoh <nama1> <nama2>\nContoh: .cekjodoh Budi Ani")
		return
	}

	name1 := args[0]
	name2 := strings.Join(args[1:], " ")

	combined := strings.ToLower(name1) + "+" + strings.ToLower(name2)
	hasher := fnv.New32a()
	hasher.Write([]byte(combined))
	percentage := int(hasher.Sum32()) % 101

	var comment string
	switch {
	case percentage >= 90:
		comment = "💕 Wah, kalian jodoh banget! Langsung nikah aja!"
	case percentage >= 70:
		comment = "😍 Cocok banget nih! Tinggal minta restu ortu~"
	case percentage >= 50:
		comment = "😊 Lumayan cocok, masih bisa diperjuangkan!"
	case percentage >= 30:
		comment = "😅 Hmm, perlu usaha lebih nih..."
	case percentage >= 10:
		comment = "😬 Kayaknya kurang cocok deh..."
	default:
		comment = "💔 Maaf, sepertinya bukan jodoh..."
	}

	reply := fmt.Sprintf(
		"💘 *Cek Jodoh*\n\n👤 %s ❤️ %s\n\n📊 Kecocokan: *%d%%*\n\n%s",
		name1, name2, percentage, comment,
	)
	utils.ReplyText(client, evt, reply)
}

// HandleRate gives a random 1-100 rating for anything.
func (h *FunHandler) HandleRate(client *whatsmeow.Client, evt *events.Message, rawArgs string) {
	subject := strings.TrimSpace(rawArgs)
	if subject == "" {
		utils.ReplyText(client, evt, "⚠️ Penggunaan: .rate <sesuatu>\nContoh: .rate skripsi gw")
		return
	}

	score := rand.Intn(101)

	var emoji, comment string
	switch {
	case score >= 90:
		emoji = "🌟"
		comment = "LUAR BIASA! Sempurna!"
	case score >= 70:
		emoji = "😎"
		comment = "Mantap, keren banget!"
	case score >= 50:
		emoji = "😊"
		comment = "Lumayan sih, nggak buruk~"
	case score >= 30:
		emoji = "😅"
		comment = "Yaa... bisa lebih baik lagi..."
	case score >= 10:
		emoji = "😬"
		comment = "Aduh, kurang nih..."
	default:
		emoji = "💀"
		comment = "Parah... nggak ada harapan."
	}

	bar := strings.Repeat("█", score/10) + strings.Repeat("░", 10-score/10)

	reply := fmt.Sprintf(
		"%s *Rate*\n\n📝 %s\n\n%s %d/100\n\n%s",
		emoji, subject, bar, score, comment,
	)
	utils.ReplyText(client, evt, reply)
}

// HandleRoast sends a random funny roast (friendly Indonesian humor).
func (h *FunHandler) HandleRoast(client *whatsmeow.Client, evt *events.Message, rawArgs string) {
	name := strings.TrimSpace(rawArgs)
	if name == "" {
		name = evt.Info.Sender.User
	}

	roasts := []string{
		"Mukanya kayak Wi-Fi gratisan, semua orang connect tapi nggak ada yang mau bayar.",
		"Kalau kamu jadi makanan, paling jadi nasi putih doang. Plain banget.",
		"Otaknya sih encer, tapi sayangnya bocor.",
		"Kamu tuh kayak tugas kuliah, nggak ada yang mau ngerjain.",
		"Muka 404 Not Found. Sorry, kegantengan tidak ditemukan.",
		"Kamu kayak kode tanpa dokumentasi, nggak ada yang bisa ngerti.",
		"Nilai IP kamu kalah sama harga gorengan.",
		"Kamu tuh kayak file ZIP, harus di-extract dulu baru ada isinya... eh ternyata corrupt.",
		"Kamu kayak browser Internet Explorer, selalu ketinggalan.",
		"Mending jadi NPC aja, soalnya skenario hidup kamu nggak ada plot-nya.",
		"Kamu tuh kayak printer, cuma berfungsi kalau dimarahin dulu.",
		"Kamu kayak charger KW, connect-nya lama, charge-nya nggak nambah.",
		"Kamu tuh kayak alarm pagi, annoying tapi tetep di-snooze.",
		"Kalau hidup kamu jadi film, pasti langsung di-skip penonton.",
		"Kamu kayak PowerPoint, cuma bagus di tampilan tapi isinya kosong.",
		"Kamu tuh kayak bug di production, nggak ada yang mau tanggung jawab.",
		"Kamu kayak capslock, selalu teriak tapi nggak penting.",
		"Muka kamu kayak error 500, internal server teriak minta tolong.",
		"Kamu tuh kayak semicolon di Python, nggak dibutuhin.",
		"Kamu kayak commit tanpa message, ada tapi nggak jelas ngapain.",
	}

	roast := roasts[rand.Intn(len(roasts))]

	reply := fmt.Sprintf("🔥 *Roasting Time!*\n\n👤 %s\n\n%s", name, roast)
	utils.ReplyText(client, evt, reply)
}

// HandleSiapaDia randomly picks a group member to answer a question.
func (h *FunHandler) HandleSiapaDia(client *whatsmeow.Client, evt *events.Message, rawArgs string) {
	question := strings.TrimSpace(rawArgs)
	if question == "" {
		utils.ReplyText(client, evt, "⚠️ Penggunaan: .siapadia <pertanyaan>\nContoh: .siapadia yang paling rajin")
		return
	}

	if !evt.Info.IsGroup {
		utils.ReplyText(client, evt, "⚠️ Command ini hanya bisa dipakai di grup.")
		return
	}

	groupInfo, err := client.GetGroupInfo(context.Background(), evt.Info.Chat)
	if err != nil {
		utils.ReplyText(client, evt, "❌ Gagal mendapatkan info grup.")
		return
	}

	if len(groupInfo.Participants) == 0 {
		utils.ReplyText(client, evt, "❌ Tidak ada anggota dalam grup.")
		return
	}

	picked := groupInfo.Participants[rand.Intn(len(groupInfo.Participants))]

	reply := fmt.Sprintf(
		"🎯 *Siapa Dia?*\n\n❓ %s\n\n👉 Jawabannya adalah: *@%s*!",
		question, picked.JID.User,
	)
	utils.ReplyTextWithMentions(client, evt, reply, []string{picked.JID.String()})
}

// HandleSeberapa gives a deterministic percentage for "seberapa X nama".
func (h *FunHandler) HandleSeberapa(client *whatsmeow.Client, evt *events.Message, rawArgs string) {
	rawArgs = strings.TrimSpace(rawArgs)
	if rawArgs == "" {
		utils.ReplyText(client, evt, "⚠️ Penggunaan: .seberapa <sifat> <nama>\nContoh: .seberapa ganteng Budi")
		return
	}

	hasher := fnv.New32a()
	hasher.Write([]byte(strings.ToLower(rawArgs)))
	percentage := int(hasher.Sum32()) % 101

	var emoji string
	switch {
	case percentage >= 80:
		emoji = "🔥🔥🔥"
	case percentage >= 60:
		emoji = "😎"
	case percentage >= 40:
		emoji = "🤔"
	case percentage >= 20:
		emoji = "😅"
	default:
		emoji = "💀"
	}

	bar := strings.Repeat("█", percentage/10) + strings.Repeat("░", 10-percentage/10)

	reply := fmt.Sprintf(
		"📊 *Seberapa %s?*\n\n%s %d%%\n\n%s",
		rawArgs, bar, percentage, emoji,
	)
	utils.ReplyText(client, evt, reply)
}


