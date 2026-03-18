package org.indis.app.data.repository

import org.indis.app.ui.wallet.CredentialCard

interface CredentialRepository {
    suspend fun listCredentials(): List<CredentialCard>
}
