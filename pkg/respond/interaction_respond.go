package respond

import "github.com/bwmarrin/discordgo"

func InteractionRespond(s *discordgo.Session, i *discordgo.InteractionCreate, content string, ephemeral bool) error {
	var flags discordgo.MessageFlags
	if ephemeral {
		flags = discordgo.MessageFlagsEphemeral
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   flags,
		},
	})
	if err != nil {
		return err
	}
	return nil
}
