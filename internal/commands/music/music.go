package music

import (
	"log/slog"
	"math/rand"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/JamesTiberiusKirk/DJGary/internal/discord"

	"github.com/bwmarrin/discordgo"
)

func StartRoutine() {
	SongSignal = make(chan PkgSong)
	go globalPlay(SongSignal)
}

func joinVoiceChannel(v *VoiceInstance, s *discordgo.Session, m *discordgo.MessageCreate) *VoiceInstance {
	voiceChannelID := discord.SearchVoiceChannelByUserID(m.Author.ID)
	if voiceChannelID == "" {
		slog.Warn("Voice channel id not found when trying to join voice channel")
		discord.SendChannelMessage(m.ChannelID, "**[Music]** You need to join a voice channel first!")
		return nil
	}

	if v != nil {
		slog.Info("voice instance already created")
	} else {
		guildID := discord.SearchGuildByChannelID(m.ChannelID)
		mutex.Lock()
		v = new(VoiceInstance)
		VoiceInstances[guildID] = v
		v.GuildID = guildID
		v.session = s
		mutex.Unlock()
	}

	var err error
	v.voice, err = discord.JoinVoiceChannel(v.GuildID, voiceChannelID, false, true)
	if err != nil {
		v.Stop()
		return nil
	}

	err = v.voice.Speaking(false)
	if err != nil {
		slog.Warn("failed to speak in voice channel", "error", err)
		return nil
	}

	slog.Info("new voice instance created")
	return v
}

func LeaveVoice(v *VoiceInstance, m *discordgo.MessageCreate) {
	if v == nil {
		slog.Info("unable to leave voice channel when bot is not in one")
		return
	}

	v.Stop()
	time.Sleep(150 * time.Millisecond)

	err := v.voice.Disconnect()
	if err != nil {
		slog.Warn("failed to leave voice channel", "error", err)
		return
	}

	slog.Info("voice channel left")
	mutex.Lock()
	delete(VoiceInstances, v.GuildID)
	mutex.Unlock()
	discord.SendChannelMessage(m.ChannelID, "**[Music]** Voice channel left!")
}

func PlayMusic(n []string, v *VoiceInstance, s *discordgo.Session, m *discordgo.MessageCreate) {
	if v == nil {
		slog.Info("bot is not in a voice channel, joining now", "userId", m.Author.ID, "username", m.Author.Username)
		v = joinVoiceChannel(v, s, m)

		if v == nil {
			slog.Warn("failed to join voice channel", "userId", m.Author.ID, "username", m.Author.Username)
			return
		}
	}

	voiceChannelID := discord.SearchVoiceChannelByUserID(m.Author.ID)
	if v.voice.ChannelID != voiceChannelID {
		discord.SendChannelMessage(m.ChannelID, "**[Music]** <@"+m.Author.ID+"> you need to join my voice channel first!")
		return
	}

	var videos []*Video

	shuffle := false
	if n[0] == "shuffle" {
		shuffle = true
		n = n[1:]
	}

	// If a youtube url is sent as argument
	if IsYoutubeURL(n[0]) {
		urlParser, err := url.Parse(n[0])
		if err != nil {
			slog.Warn("failed to parse YouTube url", "error", err)
			discord.SendChannelMessage(m.ChannelID, "**[Music]** Oops, something wrong happened when parsing youtube url")
			return
		}

		query := urlParser.Query()
		urlVideoID := query.Get("v")
		urlPlayListID := query.Get("list")

		if urlVideoID != "" {
			ytv, err := findVideoByVideoID(urlVideoID)
			if err != nil {
				slog.Info("failed to find video on YouTube from videoID", "videoID", urlVideoID, "error", err)
				discord.SendChannelMessage(m.ChannelID, "**[Music]** Oops, something went wrong when fetching title on YouTube")
				return
			}
			videos = append(videos, ytv)
		}

		if urlPlayListID != "" {
			ytvs, err := findPlaylistByPlaylistID(urlPlayListID)
			if err != nil {
				slog.Info("failed to find videos on YouTube from playlistID", "playlistID", urlPlayListID, "error", err)
				discord.SendChannelMessage(m.ChannelID, "**[Music]** Oops, something went wrong when fetching title on YouTube")
				return
			}
			slog.Info("found videos on YouTube from playlistID", "playlistID", urlPlayListID, "videos", ytvs, "len", len(ytvs))
			videos = append(videos, ytvs...)
		}

		// If argument(s) is not a youtube url
	} else {
		name := strings.Join(n, " ")
		var err error
		ytv, err := SearchVideoByName(name)
		if err != nil {
			slog.Info("failed to find song by searching YouTube", "name", name, "error", err)
			discord.SendChannelMessage(m.ChannelID, "**[Music]** Can't find a song with that name")
			return
		}
		videos = append(videos, ytv)
	}

	if shuffle {
		for i := range videos {
			j := rand.Intn(i + 1)
			videos[i], videos[j] = videos[j], videos[i]
		}
	}

	for _, video := range videos {
		song, err := getSongDataByVideo(video, v, m)
		if err != nil || song.data.ID == "" {
			if err != nil {
				slog.Warn("failed to get song data through youtube-dl", "error", err)
			}

			discord.SendChannelMessage(m.ChannelID, "**[Music]** Unable to find song")
			return
		}

		// discord.SendChannelMessage(m.ChannelID, "**[Music]** "+song.data.UserName+" has added **"+song.data.Title+"** to the queue")

		go func() {
			SongSignal <- song
		}()
	}
	discord.SendChannelMessage(m.ChannelID, "**[Music]** "+" Queued up "+strconv.Itoa(len(videos))+" songs")
}

func SkipMusic(v *VoiceInstance, m *discordgo.MessageCreate) {
	slog.Info("user is skipping song", "userId", m.Author.ID, "username", m.Author.Username)

	if v == nil {
		slog.Info("failed to skip song, bot is not in a voice channel", "userId", m.Author.ID, "username", m.Author.Username)
		discord.SendChannelMessage(m.ChannelID, "**[Music]** Can't skip song when not in a voice channel")
		return
	}

	if len(v.queue) == 0 {
		slog.Info("failed to skip song, the queue is empty", "userId", m.Author.ID, "username", m.Author.Username)
		discord.SendChannelMessage(m.ChannelID, "**[Music]** Can't skip song when not playing")
		return
	}

	if v.Skip() {
		slog.Info("failed to skip song, current song is paused", "userId", m.Author.ID, "username", m.Author.Username)
		discord.SendChannelMessage(m.ChannelID, "**[Music]** Can't skip song when current is paused")
	} else {
		slog.Info("successfully skipped song", "userId", m.Author.ID, "username", m.Author.Username)
	}
}

func StopMusic(v *VoiceInstance, m *discordgo.MessageCreate) {
	slog.Info("user is stopping music", "userId", m.Author.ID, "username", m.Author.Username)

	if v == nil {
		slog.Info("failed to stop music, bot is not in a voice channel", "userId", m.Author.ID, "username", m.Author.Username)
		discord.SendChannelMessage(m.ChannelID, "**[Music]** Can't stop playing music when not in a voice channel!")
		return
	}
	voiceChannelID := discord.SearchVoiceChannelByUserID(m.Author.ID)
	if v.voice.ChannelID != voiceChannelID {
		slog.Info("failed to stop music, user is not in same voice channel as bot", "userId", m.Author.ID, "username", m.Author.Username)
		discord.SendChannelMessage(m.ChannelID, "**[Music]** <@"+m.Author.ID+"> You need to join my voice channel to stop music!")
		return
	}

	v.Stop()
	slog.Info("successfully stopped playing music")
	discord.SendChannelMessage(m.ChannelID, "**[Music]** I have now stopped playing music!")
}
