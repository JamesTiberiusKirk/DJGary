package discordbot

//	This function replaces spaces in a string slice with "_"
func replaceSpace(s []string) string {
	if len(s) > 1 { // Checks if name is more than one word
		var result string //	String to be returned
		
		for i := 0; i < len(s); i++ { // Loops through slice and adds index
			result += s[i]

			if i != len(s)-1 { // If current index isn't last, "_" is appended
				result += "_"
			}
		}

		return result
	} else { // If name is 1 word, name is set
		return s[0]
	}
}


// searchGuild search the guild ID
func searchGuild(textChannelID string) (guildID string) {
	channel, _ := dg.Channel(textChannelID)
	guildID = channel.GuildID
	return guildID
}

// searchVoiceChannel search the voice channel id into from guild.
func searchVoiceChannel(user string) (voiceChannelID string) {
	for _, g := range dg.State.Guilds {
		for _, v := range g.VoiceStates {
			if v.UserID == user {
				return v.ChannelID
			}
		}
	}
	return ""
}
